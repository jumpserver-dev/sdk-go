package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const s3UploadPartSize int64 = 64 * 1024 * 1024

type S3ReplayStorage struct {
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

func (s S3ReplayStorage) Upload(gZipFilePath, target string) (err error) {

	file, err := os.Open(gZipFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("S3 upload requires a regular file: %s", gZipFilePath)
	}
	ctx := context.Background()
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(s.Region),
	}
	if s.AccessKey != "" && s.SecretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(s.AccessKey, s.SecretKey, "")))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		// Preserve compatibility with S3 endpoints without optional checksum support.
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		if s.Endpoint != "" {
			endpoint := s.Endpoint
			if !strings.Contains(endpoint, "://") {
				endpoint = "https://" + endpoint
			}
			o.BaseEndpoint = aws.String(endpoint)
		}
	})
	if info.Size() <= s3UploadPartSize {
		_, err = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:        aws.String(s.Bucket),
			Key:           aws.String(target),
			Body:          io.NewSectionReader(file, 0, info.Size()),
			ContentLength: aws.Int64(info.Size()),
		})
		return err
	}
	return s.uploadMultipart(ctx, client, file, target, info.Size())
}

func (s S3ReplayStorage) uploadMultipart(ctx context.Context, client *s3.Client, file *os.File, target string, size int64) (err error) {
	// Grow parts for large files without exceeding S3's 10,000-part limit.
	partSize := max(s3UploadPartSize, (size-1)/10000+1)
	if partSize > 5*1024*1024*1024 {
		return errors.New("file exceeds the S3 multipart upload size limit")
	}
	partCount := int((size-1)/partSize + 1)
	upload, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(target),
	})
	if err != nil {
		return err
	}
	if aws.ToString(upload.UploadId) == "" {
		return errors.New("S3 multipart upload returned an empty upload ID")
	}
	defer func() {
		if err == nil {
			return
		}
		abortCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, abortErr := client.AbortMultipartUpload(abortCtx, &s3.AbortMultipartUploadInput{
			Bucket: aws.String(s.Bucket), Key: aws.String(target), UploadId: upload.UploadId,
		})
		if abortErr != nil {
			err = errors.Join(err, fmt.Errorf("abort multipart upload %s: %w", *upload.UploadId, abortErr))
		}
	}()

	uploadCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	parts := make([]types.CompletedPart, partCount)
	jobs := make(chan int)
	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error
	for range min(5, partCount) {
		wg.Go(func() {
			for index := range jobs {
				if uploadCtx.Err() != nil {
					return
				}
				offset := int64(index) * partSize
				length := min(partSize, size-offset)
				part, uploadErr := client.UploadPart(uploadCtx, &s3.UploadPartInput{
					Bucket:        aws.String(s.Bucket),
					Key:           aws.String(target),
					UploadId:      upload.UploadId,
					PartNumber:    aws.Int32(int32(index + 1)),
					Body:          io.NewSectionReader(file, offset, length),
					ContentLength: aws.Int64(length),
				})
				if uploadErr == nil && aws.ToString(part.ETag) == "" {
					uploadErr = errors.New("S3 upload part returned an empty ETag")
				}
				if uploadErr != nil {
					once.Do(func() {
						firstErr = fmt.Errorf("upload part %d: %w", index+1, uploadErr)
						cancel()
					})
					return
				}
				parts[index] = types.CompletedPart{ETag: part.ETag, PartNumber: aws.Int32(int32(index + 1))}
			}
		})
	}
sendParts:
	for index := range partCount {
		select {
		case jobs <- index:
		case <-uploadCtx.Done():
			break sendParts
		}
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket: aws.String(s.Bucket), Key: aws.String(target), UploadId: upload.UploadId,
		MultipartUpload: &types.CompletedMultipartUpload{Parts: parts},
	})
	return err
}

func (s S3ReplayStorage) TypeName() string {
	return "s3"
}

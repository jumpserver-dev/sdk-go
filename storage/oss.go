package storage

import (
	"errors"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type OSSReplayStorage struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
}

func (o OSSReplayStorage) Upload(gZipFilePath, target string) (err error) {
	// 创建OSSClient实例。如果 endpoint 是空，会造成 panic，所以需要检查一下
	if o.Endpoint == "" {
		return ErrEmptyEndpoint
	}
	client, err := oss.New(o.Endpoint, o.AccessKey, o.SecretKey)
	if err != nil {
		return
	}
	bucket, err := client.Bucket(o.Bucket)
	if err != nil {
		return err
	}
	return bucket.PutObjectFromFile(target, gZipFilePath)
}

func (o OSSReplayStorage) TypeName() string {
	return "oss"
}

var ErrEmptyEndpoint = errors.New("oss endpoint is empty")

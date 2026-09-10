package storage

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestS3ReplayStorageUpload(t *testing.T) {
	dir := setupS3TestEnv(t)
	filename := filepath.Join(dir, "replay.gz")
	const content = "recorded session"
	if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	for _, explicit := range []bool{true, false} {
		name, accessKey, token := "environment", "env-access", "env-token"
		if explicit {
			name, accessKey, token = "explicit", "config-access", ""
		}
		t.Run(name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodPut || r.URL.Path != "/replays/2026/session.gz" {
					t.Errorf("unexpected upload: %s %s", r.Method, r.URL.Path)
				}
				auth := r.Header.Get("Authorization")
				if !strings.Contains(auth, "Credential="+accessKey+"/") || !strings.Contains(auth, "/us-east-1/s3/aws4_request") {
					t.Errorf("unexpected signing credentials or region: %s", auth)
				}
				if r.Header.Get("X-Amz-Security-Token") != token {
					t.Error("unexpected session token")
				}
				if r.Header.Get("X-Amz-Sdk-Checksum-Algorithm") != "" || r.Header.Get("X-Amz-Trailer") != "" {
					t.Error("upload requires optional S3 checksum support")
				}
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != content {
					t.Errorf("unexpected upload body: %q, error: %v", body, err)
				}
				w.Header().Set("ETag", `"replay-etag"`)
			}))
			defer server.Close()

			storage := S3ReplayStorage{Bucket: "replays", Region: "us-east-1", Endpoint: server.URL}
			if explicit {
				storage.AccessKey, storage.SecretKey = accessKey, "config-secret"
			}
			if err := storage.Upload(filename, "2026/session.gz"); err != nil {
				t.Fatal(err)
			}
			if requests.Load() != 1 {
				t.Fatalf("expected one upload, got %d", requests.Load())
			}
		})
	}
}

func TestS3ReplayStorageMultipartUpload(t *testing.T) {
	dir := setupS3TestEnv(t)
	file, err := os.Create(filepath.Join(dir, "replay.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	const size = s3UploadPartSize + 1024*1024
	if err := file.Truncate(size); err != nil {
		t.Fatal(err)
	}
	for index := range 2 {
		if _, err := file.WriteAt([]byte(fmt.Sprintf("part-%d", index+1)), int64(index)*s3UploadPartSize); err != nil {
			t.Fatal(err)
		}
	}
	for _, failure := range []string{"", "part", "complete"} {
		t.Run("failure="+failure, func(t *testing.T) {
			var aborts, completes, firstPartAttempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				if r.URL.Path != "/replays/session.gz" {
					t.Errorf("unexpected upload path: %s", r.URL.Path)
				}
				if r.Header.Get("X-Amz-Sdk-Checksum-Algorithm") != "" || r.Header.Get("X-Amz-Trailer") != "" {
					t.Error("multipart upload requires optional S3 checksum support")
				}
				query := r.URL.Query()
				if !query.Has("uploads") && query.Get("uploadId") != "upload" {
					t.Errorf("unexpected upload ID: %s", query.Get("uploadId"))
				}
				switch {
				case r.Method == http.MethodPost && query.Has("uploads"):
					_, _ = io.WriteString(w, `<InitiateMultipartUploadResult><UploadId>upload</UploadId></InitiateMultipartUploadResult>`)
				case r.Method == http.MethodPut:
					if failure == "part" {
						w.WriteHeader(http.StatusForbidden)
						_, _ = io.WriteString(w, `<Error><Code>AccessDenied</Code></Error>`)
						return
					}
					number, err := strconv.Atoi(query.Get("partNumber"))
					if err != nil || number < 1 || number > 2 {
						t.Errorf("unexpected part number: %s", query.Get("partNumber"))
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					prefix := make([]byte, len("part-1"))
					if _, err := io.ReadFull(r.Body, prefix); err != nil || string(prefix) != fmt.Sprintf("part-%d", number) {
						t.Errorf("incorrect part offset: %q, error: %v", prefix, err)
					}
					n, err := io.Copy(io.Discard, r.Body)
					want := min(s3UploadPartSize, size-int64(number-1)*s3UploadPartSize)
					if err != nil || n+int64(len(prefix)) != want || r.ContentLength != want {
						t.Errorf("incorrect part length: %d, expected %d, error: %v", n+int64(len(prefix)), want, err)
					}
					// Force a retry after reading the body to verify that the section is rewound.
					if failure == "" && number == 1 && firstPartAttempts.Add(1) == 1 {
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = io.WriteString(w, `<Error><Code>InternalError</Code></Error>`)
						return
					}
					w.Header().Set("ETag", fmt.Sprintf(`"part-%d"`, number))
				case r.Method == http.MethodPost && query.Has("uploadId"):
					completes.Add(1)
					var body struct {
						Parts []struct {
							Number int `xml:"PartNumber"`
							ETag   string
						} `xml:"Part"`
					}
					if err := xml.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Parts) != 2 {
						t.Errorf("invalid completion request: %+v, error: %v", body, err)
					}
					for index, part := range body.Parts {
						if part.Number != index+1 || part.ETag != fmt.Sprintf(`"part-%d"`, index+1) {
							t.Errorf("parts completed out of order: %+v", body.Parts)
						}
					}
					if failure == "complete" {
						w.WriteHeader(http.StatusBadRequest)
						_, _ = io.WriteString(w, `<Error><Code>InvalidPart</Code></Error>`)
						return
					}
					_, _ = io.WriteString(w, `<CompleteMultipartUploadResult><ETag>"complete"</ETag></CompleteMultipartUploadResult>`)
				case r.Method == http.MethodDelete:
					aborts.Add(1)
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			storage := S3ReplayStorage{Bucket: "replays", Region: "us-east-1", Endpoint: server.URL}
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			err := storage.Upload(file.Name(), "session.gz")
			runtime.ReadMemStats(&after)
			if (err != nil) != (failure != "") {
				t.Fatalf("unexpected upload error: %v", err)
			}
			wantAborts, wantCompletes := int32(0), int32(1)
			if failure != "" {
				wantAborts = 1
			}
			if failure == "part" {
				wantCompletes = 0
			}
			if aborts.Load() != wantAborts || completes.Load() != wantCompletes {
				t.Fatalf("aborts=%d, completes=%d", aborts.Load(), completes.Load())
			}
			if failure == "" {
				if firstPartAttempts.Load() != 2 {
					t.Fatalf("expected a retried part, got %d attempts", firstPartAttempts.Load())
				}
				if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 32*1024*1024 {
					t.Fatalf("multipart upload buffered file contents: allocated %d bytes", allocated)
				}
			}
		})
	}
}

func setupS3TestEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "env-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "env-secret")
	t.Setenv("AWS_SESSION_TOKEN", "env-token")
	t.Setenv("AWS_REQUEST_CHECKSUM_CALCULATION", "when_supported")
	t.Setenv("AWS_MAX_ATTEMPTS", "2")
	return dir
}

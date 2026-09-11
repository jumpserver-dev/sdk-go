package storage

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/jumpserver-dev/sdk-go/model"
)

func TestBulkSaveEs(t *testing.T) {
	var bulkRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		if r.Method == http.MethodGet && r.URL.Path == "/" {
			_, _ = w.Write([]byte(`{"version":{"number":"7.17.10"}}`))
			return
		}
		bulkRequests.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/commands/_doc/_bulk" {
			t.Errorf("unexpected bulk request: %s %s", r.Method, r.URL.Path)
		}
		decoder := json.NewDecoder(r.Body)
		var action map[string]json.RawMessage
		if err := decoder.Decode(&action); err != nil || action["index"] == nil {
			t.Errorf("unexpected bulk action: %v, error: %v", action, err)
		}
		var command model.Command
		if err := decoder.Decode(&command); err != nil || command.Input != "whoami" {
			t.Errorf("unexpected command: %q, error: %v", command.Input, err)
		}
		_, _ = w.Write([]byte(`{"errors":false,"items":[{"index":{"status":201}}]}`))
	}))
	defer server.Close()

	storage := ESCommandStorage{Hosts: []string{server.URL}, Index: "commands", DocType: "_doc"}
	if err := storage.BulkSaveEs([]*model.Command{{Input: "whoami"}}); err != nil {
		t.Fatal(err)
	}
	if bulkRequests.Load() != 1 {
		t.Fatalf("expected one bulk request, got %d", bulkRequests.Load())
	}
}

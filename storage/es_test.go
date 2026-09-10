package storage

import (
	"bytes"
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

func TestEsIndexResponse(t *testing.T) {
	respBodys := [][2]string{
		{"index", `{"took":24,"errors":false,"items":[{"index":{"_index":"jumpserver-test-1","_type":"_doc","_id":"mo9R9IkBIDTIizd_N0BL","_version":1,"result":"created","_shards":{"total":1,"successful":1,"failed":0},"_seq_no":3,"_primary_term":1,"status":201}},{"index":{"_index":"jumpserver-test-1","_type":"_doc","_id":"m49R9IkBIDTIizd_N0BL","_version":1,"result":"created","_shards":{"total":1,"successful":1,"failed":0},"_seq_no":4,"_primary_term":1,"status":201}}]}`},
		{"create", `{"took":36,"errors":false,"items":[{"create":{"_index":"jumpserver-test-1","_type":"_doc","_id":"mI9Q9IkBIDTIizd_5UBF","_version":1,"result":"created","_shards":{"total":1,"successful":1,"failed":0},"_seq_no":1,"_primary_term":1,"status":201}},{"create":{"_index":"jumpserver-test-1","_type":"_doc","_id":"mY9Q9IkBIDTIizd_5UBF","_version":1,"result":"created","_shards":{"total":1,"successful":1,"failed":0},"_seq_no":2,"_primary_term":1,"status":201}}]}`},
	}

	for idx := range respBodys {
		action, data := respBodys[idx][0], respBodys[idx][1]
		var (
			blk  *bulkResponse
			body bytes.Buffer = *bytes.NewBufferString(data)
		)

		if err := json.NewDecoder(&body).Decode(&blk); err != nil {
			t.Fatalf("ES failure to parse response body: %s", err)
		} else {
			for _, d := range blk.Items {
				if _, ok := d[action]; !ok {
					t.Fatalf("can not get action response from es bulk response body %d", idx)
				}
			}
		}
	}
}

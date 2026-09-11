package httplib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jumpserver-dev/sdk-go/model"
)

func TestWithResponseIsolation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		cookie, err := r.Cookie("scope")
		if err != nil || cookie.Value != name || r.Header.Get("X-Scope") != name || r.Header.Get("Authorization") != "Bearer "+name {
			t.Errorf("client settings were lost or shared: %v %v", r.Header, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"asset","future":9007199254740993,"optional":null}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, 0)
	if err != nil {
		t.Fatal(err)
	}
	client.SetHeader("X-Scope", "base")
	client.SetCookie("scope", "base")
	client.SetAuthSign(&CustomAuth{AuthScheme: "Bearer", Token: "base"})
	var fields struct {
		Future   int64 `json:"future"`
		Missing  *bool `json:"missing"`
		Optional *bool `json:"optional"`
	}
	bound := client.WithResponse(&fields)
	bound.SetHeader("X-Scope", "bound")
	bound.SetCookie("scope", "bound")
	bound.SetAuthSign(&CustomAuth{AuthScheme: "Bearer", Token: "bound"})
	var asset model.Asset
	if _, err = bound.Get("/bound", &asset); err != nil {
		t.Fatal(err)
	}
	if asset.ID != "asset" || fields.Future != 9007199254740993 || fields.Missing != nil || fields.Optional != nil {
		t.Fatal("typed or extra fields were decoded incorrectly")
	}
	fields.Future = 0
	if _, err = client.Get("/base", &asset); err != nil || fields.Future != 0 {
		t.Fatalf("response target leaked into the original client: %v", err)
	}
}

func TestHandleResponseFields(t *testing.T) {
	request, err := http.NewRequest(http.MethodPost, "http://localhost/upload", nil)
	if err != nil {
		t.Fatal(err)
	}
	response := &http.Response{
		StatusCode: http.StatusOK,
		Request:    request,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"asset","future_result":true}`)),
	}
	var asset model.Asset
	var fields struct {
		Result bool `json:"future_result"`
	}
	if err = (&Client{}).WithResponse(&fields).handleResp(response, &asset); err != nil {
		t.Fatal(err)
	}
	if asset.ID != "asset" {
		t.Fatal("typed response field was lost")
	}
	if !fields.Result {
		t.Fatal("upload response lost new fields")
	}
}

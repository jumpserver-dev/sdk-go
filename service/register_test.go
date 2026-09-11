package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTerminalRegistrationOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != TerminalRegisterURL || r.Header.Get("Authorization") != "BootstrapToken bootstrap" {
			t.Error("registration URL or authentication changed")
		}
		var data map[string]any
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			t.Error(err)
		}
		if data["name"] != "terminal" || data["type"] != "panda" {
			t.Errorf("registration fields changed: %v", data)
		}
		if data["provider_id"] != "provider" || data["future_setting"] != true {
			t.Errorf("registration options lost: %v", data)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"name": "terminal", "provider_id": data["provider_id"]})
	}))
	defer server.Close()
	svc, err := NewAuthJMService(JMSCoreHost(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	var fields struct {
		ProviderID string `json:"provider_id"`
	}
	options := TerminalRegistrationOptions{
		ProviderID: "provider", ExtraFields: map[string]any{"future_setting": true},
	}
	terminal, err := RegisterTerminalAccountWithOptions(server.URL, "panda", "terminal", "bootstrap", options)
	if err != nil {
		t.Fatal(err)
	}
	if terminal.Name != "terminal" {
		t.Fatal("standalone registration response was lost")
	}
	terminal, err = svc.WithResponse(&fields).RegisterTerminalWithOptions("terminal", "bootstrap", "panda", options)
	if err != nil {
		t.Fatal(err)
	}
	if terminal.Name != "terminal" || fields.ProviderID != "provider" {
		t.Fatal("service registration did not send provider_id")
	}
}

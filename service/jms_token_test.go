package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConnectTokenFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" || r.Header.Get(orgHeaderKey) != orgHeaderValue {
			t.Error("SDK authentication or organization header was lost")
		}
		w.Header().Set("Content-Type", "application/json")
		var data map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			t.Error(err)
		}
		switch r.URL.Path {
		case SuperConnectTokenSecretURL:
			if string(data["id"]) != `"token"` || string(data["expire_now"]) != "false" || string(data["future_request"]) != "true" {
				t.Errorf("incorrect secret request: %s", data)
			}
			_, _ = w.Write([]byte(`{"id":"token","connect_options":{"remote_microphone":false,"rdp_connection_speed":"wan","future_option":true},"asset":{"spec_info":{"success_selector":"css=#done","interactive_selector":"css=#mfa","allowed_urls":["https://example.com"]}}}`))
		case SuperConnectTokenInfoURL:
			if r.URL.Query().Get("create_ticket") != "true" || r.Header.Get(svcHeader) == "" {
				t.Error("creation query/signature was lost")
			}
			for key, value := range map[string]string{"input_secret_type": `"ssh_key"`, "save_personal_credential": "false", "future_request": "true", "user": `"user"`} {
				if string(data[key]) != value {
					t.Errorf("request field %s: %s", key, data[key])
				}
			}
			if _, ok := data["personal_credential_version"]; ok {
				t.Error("absent optional field was submitted")
			}
			_, _ = w.Write([]byte(`{"id":"token","user":{"id":"user","name":"User"},"asset":{"id":"asset","name":"Host"},"account":"@INPUT","is_active":false,"face_token":"face","personal_credential_id":null,"date_expired":null,"future_response":{"enabled":true}}`))
		case SuperConnectTokenVirtualAppOptionURL:
			_, _ = w.Write([]byte(`{"name":"app","provider":{"id":"provider","load":"offline","host":{"address":"192.0.2.1"},"account":{"username":"root","privileged":true},"gateway":null,"future_route":2222}}`))
		default:
			t.Errorf("unexpected URL: %s", r.URL)
		}
	}))
	defer server.Close()
	svc, err := NewAuthJMService(JMSCoreHost(server.URL), JMSAccessKey("key", "12345678901234567890123456789012"))
	if err != nil {
		t.Fatal(err)
	}
	disabled := false
	var tokenFields struct {
		Options struct {
			Enabled bool `json:"future_option"`
		} `json:"connect_options"`
	}
	token, err := svc.WithResponse(&tokenFields).GetConnectTokenInfoWithOptions("token", ConnectTokenSecretOptions{
		ExpireNow: &disabled, ExtraFields: map[string]any{"future_request": true, "id": "wrong"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if token.ConnectOptions.RemoteMicrophone == nil || *token.ConnectOptions.RemoteMicrophone || token.Asset.SpecInfo.SuccessSelector != "css=#done" {
		t.Fatal("new connection fields were lost")
	}
	if !tokenFields.Options.Enabled {
		t.Fatal("future option was lost")
	}
	var infoFields struct {
		Future struct {
			Enabled bool `json:"enabled"`
		} `json:"future_response"`
	}
	info, err := svc.WithResponse(&infoFields).CreateSuperConnectToken(&SuperConnectTokenReq{
		UserId: "user", InputSecretType: "ssh_key", SavePersonalCredential: &disabled,
		ConnectOptions: map[string]any{"use_sysdba": true},
		ExtraFields:    map[string]any{"future_request": true, "user": "wrong"},
		Params:         map[string]string{"create_ticket": "true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.Account != "@INPUT" || info.FaceToken != "face" || info.IsActive == nil || *info.IsActive || info.DateExpired != nil {
		t.Fatal("token metadata was lost")
	}
	if info.User == nil || info.Asset == nil {
		t.Fatal("Core related-object fields were lost")
	}
	if !infoFields.Future.Enabled {
		t.Fatal("creation response target was not decoded")
	}
	var appFields struct {
		Provider struct {
			Port int `json:"future_route"`
		} `json:"provider"`
	}
	app, err := svc.WithResponse(&appFields).GetConnectTokenVirtualAppOption("token")
	if err != nil || app.Provider == nil || app.Provider.Host.Address != "192.0.2.1" || !app.Provider.Account.Privileged {
		t.Fatalf("provider was lost: %+v %v", app.Provider, err)
	}
	if appFields.Provider.Port != 2222 {
		t.Fatal("future provider fields were lost")
	}
}

func TestTokenErrorResponseFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"new_error","detail":"denied","expired":true,"future_reason":"changed"}`))
	}))
	defer server.Close()
	svc, err := NewAuthJMService(JMSCoreHost(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	var fields struct {
		Reason string `json:"future_reason"`
	}
	status, err := svc.WithResponse(&fields).CheckTokenStatus("token")
	if err == nil || status.Code != "new_error" || !status.Expired {
		t.Fatal("HTTP error or decoded error fields were lost")
	}
	if fields.Reason != "changed" {
		t.Fatal("HTTP error discarded future response fields")
	}
	fields.Reason = "unchanged"
	if _, err := svc.CheckTokenStatus("token"); err == nil || fields.Reason != "unchanged" {
		t.Fatal("response target leaked into a later call")
	}
}

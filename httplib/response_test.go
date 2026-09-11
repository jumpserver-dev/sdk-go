package httplib

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/jumpserver-dev/sdk-go/model"
)

func TestResponseDecodeErrors(t *testing.T) {
	for _, body := range []string{
		`{"id":"token","expire_time":"bad","future":true}`,
		`{"id":"token","expire_time":60,"future":"bad"}`,
	} {
		t.Run(body, func(t *testing.T) {
			var token model.ConnectTokenInfo
			var extra struct {
				ID     string `json:"id"`
				Future bool   `json:"future"`
			}
			client := (&Client{}).WithResponse(&extra)
			err := json.Unmarshal([]byte(body), client.responseDestination(&token))
			var typeErr *json.UnmarshalTypeError
			if !errors.As(err, &typeErr) || token.ID != "token" || extra.ID != "token" {
				t.Fatalf("decode error discarded successful fields: %+v %+v %v", token, extra, err)
			}
			if token.ExpireTime != 60 && !extra.Future {
				t.Fatal("an error in one destination prevented decoding the other")
			}
		})
	}
}

func TestResponseWithoutSDKDestination(t *testing.T) {
	var extra struct {
		Future bool `json:"future"`
	}
	client := (&Client{}).WithResponse(&extra)
	if err := json.Unmarshal([]byte(`{"future":true}`), client.responseDestination(nil)); err != nil || !extra.Future {
		t.Fatalf("extra fields need an SDK destination: %v", err)
	}
}

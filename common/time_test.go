package common

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewJSONTime(t *testing.T) {
	want := time.Date(2026, time.September, 11, 10, 20, 30, 0, time.FixedZone("UTC+8", 8*60*60))
	data, err := json.Marshal(NewUTCTime(want))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"2026-09-11 02:20:30 +0000"` {
		t.Fatalf("unexpected UTC encoding: %s", data)
	}
	var got UTCTime
	if err = json.Unmarshal(data, &got); err != nil || !got.Equal(want) {
		t.Fatalf("round trip changed time: %v, error: %v", got, err)
	}
}

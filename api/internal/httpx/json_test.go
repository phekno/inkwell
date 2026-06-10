package httpx

import (
	"encoding/json"
	"testing"
)

func TestJSON(t *testing.T) {
	res, err := JSON(201, map[string]string{"id": "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 201 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if res.Headers["content-type"] != "application/json" {
		t.Fatalf("content-type = %q", res.Headers["content-type"])
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(res.Body), &got); err != nil {
		t.Fatalf("body not valid json: %v", err)
	}
	if got["id"] != "abc" {
		t.Fatalf("body = %v", got)
	}
}

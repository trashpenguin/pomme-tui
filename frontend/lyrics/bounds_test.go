package lyrics

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type responseTransport struct{}

func (responseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"plainLyrics":"` + strings.Repeat("a", (2<<20)+1) + `"}]`)), Header: make(http.Header)}, nil
}
func TestFetchRejectsOversizedResponse(t *testing.T) {
	previous := client
	client = &http.Client{Transport: responseTransport{}}
	defer func() { client = previous }()
	if _, err := Fetch(context.Background(), "artist", "title", 0); err == nil {
		t.Fatal("oversized lyrics accepted")
	}
}

package main

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestArtworkRejectsOversizedDimensions(t *testing.T) {
	var body bytes.Buffer
	png.Encode(&body, image.NewGray(image.Rect(0, 0, 2049, 1)))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(body.Bytes()) }))
	defer server.Close()
	if _, err := fetchArtwork(context.Background(), server.URL); err == nil {
		t.Fatal("oversized artwork accepted")
	}
}

func TestArtworkRejectsHTTPError(t *testing.T) {
	var body bytes.Buffer
	png.Encode(&body, image.NewGray(image.Rect(0, 0, 1, 1)))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500); w.Write(body.Bytes()) }))
	defer server.Close()
	if _, err := fetchArtwork(context.Background(), server.URL); err == nil {
		t.Fatal("HTTP error accepted")
	}
}

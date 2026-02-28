package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetUnwrapsData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{{"name": "repo1"}, {"name": "repo2"}},
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-token")
	var repos []map[string]string
	if err := client.Get(context.Background(), "", &repos); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("got %d repos, want 2", len(repos))
	}
}

func TestGet401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	}))
	defer server.Close()

	client := New(server.URL, "bad-token")
	var result any
	err := client.Get(context.Background(), "/repos", &result)
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestGet404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := New(server.URL, "token")
	var result any
	err := client.Get(context.Background(), "/missing", &result)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing Content-Type header")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]bool{"success": true},
		})
	}))
	defer server.Close()

	client := New(server.URL, "token")
	var result map[string]bool
	body := map[string]string{"name": "test"}
	if err := client.Post(context.Background(), "/repos", body, &result); err != nil {
		t.Fatalf("Post: %v", err)
	}
	if !result["success"] {
		t.Error("expected success=true")
	}
}

func TestHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"status":  "healthy",
				"version": "1.0.0",
				"temporal": map[string]any{"connected": true},
				"dynamodb": map[string]any{"connected": true},
			},
		})
	}))
	defer server.Close()

	client := New(server.URL, "token")
	health, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if health.Status != "healthy" {
		t.Errorf("status = %s, want healthy", health.Status)
	}
	if health.Version != "1.0.0" {
		t.Errorf("version = %s, want 1.0.0", health.Version)
	}
}

func TestConnectionError(t *testing.T) {
	client := New("http://localhost:1", "token")
	err := client.Get(context.Background(), "/health", nil)
	if err == nil {
		t.Fatal("expected connection error")
	}
}

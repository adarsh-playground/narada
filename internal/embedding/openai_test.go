package embedding

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestOpenAIEmbedsBatchInIndexOrder(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing authorization")
		}
		var request embeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != DefaultModel || request.Dimensions != DefaultDimensions || len(request.Input) != 2 {
			t.Fatalf("unexpected request: %+v", request)
		}
		vectors := [][]float32{make([]float32, DefaultDimensions), make([]float32, DefaultDimensions)}
		body, err := json.Marshal(map[string]any{"data": []any{
			map[string]any{"index": 1, "embedding": vectors[1]}, map[string]any{"index": 0, "embedding": vectors[0]},
		}, "usage": map[string]any{"prompt_tokens": 7, "total_tokens": 7}})
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}, nil
	})}
	client, err := NewOpenAI("test-key", "https://example.test/v1", DefaultModel, httpClient)
	if err != nil {
		t.Fatal(err)
	}
	vectors, err := client.Embed(context.Background(), []string{"one", "two"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 2 || len(vectors[0]) != DefaultDimensions {
		t.Fatalf("unexpected vectors")
	}
}

func TestOpenAIReportsEmbeddingUsage(t *testing.T) {
	vector := make([]float32, DefaultDimensions)
	body, _ := json.Marshal(map[string]any{"data": []any{map[string]any{"index": 0, "embedding": vector}}, "usage": map[string]any{"prompt_tokens": 9, "total_tokens": 9}})
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}, nil
	})}
	client, err := NewOpenAI("test-key", "https://example.test/v1", DefaultModel, httpClient)
	if err != nil {
		t.Fatal(err)
	}
	_, tokens, err := client.EmbedWithUsage(context.Background(), []string{"a question"})
	if err != nil || tokens != 9 {
		t.Fatalf("tokens=%d err=%v", tokens, err)
	}
}

func TestOpenAIRejectsMissingKey(t *testing.T) {
	if _, err := NewOpenAI("", "", "", nil); err == nil || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("error = %v", err)
	}
}

func TestRetryDelayUsesOpenAIMessage(t *testing.T) {
	message := "Rate limit reached. Please try again in 22.528s."
	if got, want := retryDelay("", message, 0), 23*time.Second+28*time.Millisecond; got != want {
		t.Fatalf("retry delay = %v, want %v", got, want)
	}
}

func TestRetryDelayPrefersRetryAfterHeader(t *testing.T) {
	if got, want := retryDelay("3", "Please try again in 20s", 0), 3500*time.Millisecond; got != want {
		t.Fatalf("retry delay = %v, want %v", got, want)
	}
}

func TestVectorLiteral(t *testing.T) {
	if got := VectorLiteral([]float32{0.5, -1.25}); got != "[0.5,-1.25]" {
		t.Fatalf("literal = %q", got)
	}
}

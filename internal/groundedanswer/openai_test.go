package groundedanswer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/adarsh/narada/internal/semanticsearch"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestGenerateGroundedAnswer(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/v1/responses" || request.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected request: %s", request.URL)
		}
		var body responseRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Store || !strings.Contains(body.Input, "BG 2.47") || !strings.Contains(body.Instructions, "only") {
			t.Fatalf("request was not grounded: %+v", body)
		}
		response := `{"model":"gpt-test","output":[{"type":"message","content":[{"type":"output_text","text":"Act without attachment [BG 2.47]."}]}],"usage":{"input_tokens":120,"output_tokens":12}}`
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})}
	client, err := NewOpenAI("test-key", "https://example.test/v1", "gpt-test", httpClient)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := client.Generate(context.Background(), "How should I work?", []semanticsearch.Result{{CitationLabel: "BG 2.47", Text: "Work without attachment."}})
	if err != nil {
		t.Fatal(err)
	}
	if answer.Text != "Act without attachment [BG 2.47]." || answer.InputTokens != 120 || answer.OutputTokens != 12 {
		t.Fatalf("unexpected answer: %+v", answer)
	}
}

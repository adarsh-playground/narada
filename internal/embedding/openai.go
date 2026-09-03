package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultModel      = "text-embedding-3-small"
	DefaultDimensions = 1536
)

type Provider interface {
	Embed(ctx context.Context, inputs []string) ([][]float32, error)
	Name() string
	Model() string
	Dimensions() int
}

type MeteredProvider interface {
	Provider
	EmbedWithUsage(ctx context.Context, inputs []string) ([][]float32, int, error)
}

type OpenAI struct {
	apiKey     string
	baseURL    string
	model      string
	dimensions int
	httpClient *http.Client
	onRetry    func(time.Duration, error)
}

var retryMessagePattern = regexp.MustCompile(`(?i)try again in\s+([0-9]*\.?[0-9]+)\s*(ms|s)\b`)

func NewOpenAI(apiKey, baseURL, model string, client *http.Client) (*OpenAI, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if strings.TrimSpace(model) == "" {
		model = DefaultModel
	}
	if model != DefaultModel {
		return nil, fmt.Errorf("model %q is not supported by the current vector(1536) schema", model)
	}
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	return &OpenAI{apiKey: apiKey, baseURL: strings.TrimRight(baseURL, "/"), model: model, dimensions: DefaultDimensions, httpClient: client}, nil
}

func (o *OpenAI) Name() string    { return "openai" }
func (o *OpenAI) Model() string   { return o.model }
func (o *OpenAI) Dimensions() int { return o.dimensions }

func (o *OpenAI) SetRetryNotifier(notifier func(time.Duration, error)) {
	o.onRetry = notifier
}

type embeddingRequest struct {
	Input          []string `json:"input"`
	Model          string   `json:"model"`
	Dimensions     int      `json:"dimensions"`
	EncodingFormat string   `json:"encoding_format"`
}

type embeddingResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func (o *OpenAI) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	vectors, _, err := o.EmbedWithUsage(ctx, inputs)
	return vectors, err
}

func (o *OpenAI) EmbedWithUsage(ctx context.Context, inputs []string) ([][]float32, int, error) {
	if len(inputs) == 0 {
		return nil, 0, fmt.Errorf("at least one embedding input is required")
	}
	for _, input := range inputs {
		if strings.TrimSpace(input) == "" {
			return nil, 0, fmt.Errorf("embedding input cannot be empty")
		}
	}
	body, err := json.Marshal(embeddingRequest{Input: inputs, Model: o.model, Dimensions: o.dimensions, EncodingFormat: "float"})
	if err != nil {
		return nil, 0, err
	}

	var lastErr error
	const maxAttempts = 8
	for attempt := 0; attempt < maxAttempts; attempt++ {
		var retryAfter string
		var responseMessage string
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/embeddings", bytes.NewReader(body))
		if err != nil {
			return nil, 0, err
		}
		request.Header.Set("Authorization", "Bearer "+o.apiKey)
		request.Header.Set("Content-Type", "application/json")
		response, err := o.httpClient.Do(request)
		if err != nil {
			lastErr = err
		} else {
			retryAfter = response.Header.Get("Retry-After")
			contents, readErr := io.ReadAll(io.LimitReader(response.Body, 32<<20))
			response.Body.Close()
			if readErr != nil {
				return nil, 0, readErr
			}
			var decoded embeddingResponse
			if err := json.Unmarshal(contents, &decoded); err != nil {
				return nil, 0, fmt.Errorf("decode embedding response: %w", err)
			}
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				if len(decoded.Data) != len(inputs) {
					return nil, 0, fmt.Errorf("embedding response contained %d vectors for %d inputs", len(decoded.Data), len(inputs))
				}
				sort.Slice(decoded.Data, func(i, j int) bool { return decoded.Data[i].Index < decoded.Data[j].Index })
				vectors := make([][]float32, len(decoded.Data))
				for i, item := range decoded.Data {
					if item.Index != i || len(item.Embedding) != o.dimensions {
						return nil, 0, fmt.Errorf("invalid embedding at index %d", item.Index)
					}
					vectors[i] = item.Embedding
				}
				return vectors, decoded.Usage.TotalTokens, nil
			}
			message := response.Status
			if decoded.Error != nil && decoded.Error.Message != "" {
				message = decoded.Error.Message
			}
			responseMessage = message
			lastErr = fmt.Errorf("OpenAI embeddings request failed: %s", message)
			if response.StatusCode != http.StatusTooManyRequests && response.StatusCode < 500 {
				return nil, 0, lastErr
			}
		}
		if attempt < maxAttempts-1 {
			delay := retryDelay(retryAfter, responseMessage, attempt)
			if o.onRetry != nil {
				o.onRetry(delay, lastErr)
			}
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return nil, 0, lastErr
}

func retryDelay(retryAfter, message string, attempt int) time.Duration {
	const buffer = 500 * time.Millisecond
	const maximum = 2 * time.Minute

	var delay time.Duration
	if seconds, err := strconv.ParseFloat(strings.TrimSpace(retryAfter), 64); err == nil && seconds > 0 {
		delay = time.Duration(seconds * float64(time.Second))
	} else if retryAt, err := http.ParseTime(retryAfter); err == nil {
		delay = time.Until(retryAt)
	}
	if delay <= 0 {
		if match := retryMessagePattern.FindStringSubmatch(message); len(match) == 3 {
			value, err := strconv.ParseFloat(match[1], 64)
			if err == nil {
				delay = time.Duration(value * float64(time.Second))
				if strings.EqualFold(match[2], "ms") {
					delay = time.Duration(value * float64(time.Millisecond))
				}
			}
		}
	}
	if delay <= 0 {
		delay = time.Duration(1<<min(attempt, 6)) * time.Second
	}
	if delay > maximum {
		delay = maximum
	}
	return delay + buffer
}

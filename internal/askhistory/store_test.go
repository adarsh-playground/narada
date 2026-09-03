package askhistory

import "testing"

func TestAnswerPrices(t *testing.T) {
	tests := []struct {
		model         string
		input, output int64
	}{
		{"gpt-5.6-luna", 200, 1200},
		{"gpt-5.6-terra", 2000, 12000},
		{"gpt-5.6-sol", 4000, 20000},
	}
	for _, test := range tests {
		input, output, err := answerPrices(test.model)
		if err != nil || input != test.input || output != test.output {
			t.Fatalf("%s prices=(%d,%d) err=%v", test.model, input, output, err)
		}
	}
}

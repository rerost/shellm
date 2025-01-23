package functions

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockLLM struct {
	response string
	err      error
}

func (m mockLLM) Call(message string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestCheck_Call(t *testing.T) {
	tests := []struct {
		name     string
		llm      mockLLM
		request  CheckRequest
		expected CheckResponse
		wantErr  bool
	}{
		{
			name: "valid response",
			llm: mockLLM{
				response: `{"is_valid": true, "reason": "good result"}`,
			},
			request: CheckRequest{
				Task:        "implement feature",
				CurrentStep: "write test",
				Result:      "test written",
			},
			expected: CheckResponse{
				IsValid: true,
				Reason:  "good result",
			},
			wantErr: false,
		},
		{
			name: "invalid response",
			llm: mockLLM{
				response: `{"is_valid": false, "reason": "incomplete result"}`,
			},
			request: CheckRequest{
				Task:        "implement feature",
				CurrentStep: "write test",
				Result:      "partial test",
			},
			expected: CheckResponse{
				IsValid: false,
				Reason:  "incomplete result",
			},
			wantErr: false,
		},
		{
			name: "missing task",
			llm:  mockLLM{},
			request: CheckRequest{
				CurrentStep: "write test",
				Result:      "test written",
			},
			expected: CheckResponse{
				IsValid: false,
				Reason:  "task is required",
			},
			wantErr: false,
		},
		{
			name: "missing current step",
			llm:  mockLLM{},
			request: CheckRequest{
				Task:   "implement feature",
				Result: "test written",
			},
			expected: CheckResponse{
				IsValid: false,
				Reason:  "current_step is required",
			},
			wantErr: false,
		},
		{
			name: "missing result",
			llm:  mockLLM{},
			request: CheckRequest{
				Task:        "implement feature",
				CurrentStep: "write test",
			},
			expected: CheckResponse{
				IsValid: false,
				Reason:  "result is required",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Check{LLM: tt.llm}
			result, err := c.Call(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			var response CheckResponse
			err = json.Unmarshal([]byte(result), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, response)
		})
	}
}

package functions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockCheck struct {
	response string
	err      error
}

func (m *mockCheck) Call(_ context.Context, _ CheckRequest) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestReceiveShellCommand_Run(t *testing.T) {
	tests := []struct {
		name           string
		check          *Check
		mockLLM        mockLLM
		arguments      string
		expectedOutput string
		wantErr        bool
	}{
		{
			name: "command with successful check",
			mockLLM: mockLLM{
				response: `{"is_valid": true, "reason": "command completed successfully"}`,
			},
			arguments: `{
				"command": "echo 'test'",
				"working_directory": ".",
				"task": "test task",
				"current_step": "echo test"
			}`,
			expectedOutput: "Completion Check Passed ✓",
			wantErr:        false,
		},
		{
			name: "command with failed check",
			mockLLM: mockLLM{
				response: `{"is_valid": false, "reason": "expected output not found"}`,
			},
			arguments: `{
				"command": "echo 'test'",
				"working_directory": ".",
				"task": "test task",
				"current_step": "echo test"
			}`,
			expectedOutput: "Completion Check Failed: expected output not found",
			wantErr:        false,
		},
		{
			name:  "command without check",
			check: nil,
			arguments: `{
				"command": "echo 'test'",
				"working_directory": "."
			}`,
			expectedOutput: "test",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var check *Check
			if tt.mockLLM.response != "" {
				check = &Check{
					LLM: tt.mockLLM,
				}
			}

			r := ReceiveShellCommand{
				Check: check,
			}

			result, err := r.Run(context.Background(), tt.arguments)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Contains(t, result, tt.expectedOutput)
		})
	}
}

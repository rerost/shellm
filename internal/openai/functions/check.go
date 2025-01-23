package functions

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
)

type LLM interface {
	Call(message string) (string, error)
}

type Check struct {
	LLM LLM
}

type CheckRequest struct {
	Task        string `json:"task"`
	CurrentStep string `json:"current_step"`
	Result      string `json:"result"`
}

type CheckResponse struct {
	IsValid bool   `json:"is_valid"`
	Reason  string `json:"reason"`
}

func (c *Check) Call(ctx context.Context, request CheckRequest) (string, error) {
	if request.Task == "" {
		return toJSON(CheckResponse{
			IsValid: false,
			Reason:  "task is required",
		})
	}

	if request.CurrentStep == "" {
		return toJSON(CheckResponse{
			IsValid: false,
			Reason:  "current_step is required",
		})
	}

	if request.Result == "" {
		return toJSON(CheckResponse{
			IsValid: false,
			Reason:  "result is required",
		})
	}

	// If all fields are present, call LLM to validate
	if c.LLM != nil {
		return c.LLM.Call(formatCheckMessage(request))
	}

	// Default response if no LLM is provided
	return toJSON(CheckResponse{
		IsValid: false,
		Reason:  "LLM not configured",
	})
}

func formatCheckMessage(request CheckRequest) string {
	return `Please check if the following result satisfies the current step of the task:
Task: ` + request.Task + `
Current Step: ` + request.CurrentStep + `
Result: ` + request.Result + `

Respond with a JSON object containing:
{
  "is_valid": boolean,
  "reason": string
}
`
}

func toJSON(response CheckResponse) (string, error) {
	bytes, err := json.Marshal(response)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(bytes), nil
}

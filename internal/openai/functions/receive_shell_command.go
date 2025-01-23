package functions

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	gopenai "github.com/sashabaranov/go-openai"
)

//go:embed receive_shell_command.json
var functionJson []byte

type ReceiveShellCommand struct {
	Check *Check
}

func (r ReceiveShellCommand) Name() string {
	return "receive_shell_command"
}

func (r ReceiveShellCommand) Register() gopenai.Tool {
	var function gopenai.FunctionDefinition
	if err := json.Unmarshal(functionJson, &function); err != nil {
		panic(err)
	}

	return gopenai.Tool{
		Type:     gopenai.ToolTypeFunction,
		Function: &function,
	}
}

func (r ReceiveShellCommand) Run(ctx context.Context, arguments string) (string, error) {
	type Argument struct {
		Command          string `json:"command"`
		WorkingDirectory string `json:"working_directory"`
		Task             string `json:"task"`
		CurrentStep      string `json:"current_step"`
	}
	var arg Argument
	if err := json.Unmarshal([]byte(arguments), &arg); err != nil {
		return "", errors.WithStack(err)
	}
	fmt.Println("[Local] $", arg.Command)

	tempfile, err := os.CreateTemp("", "shellm-")
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer tempfile.Close()

	if err := os.Chmod(tempfile.Name(), 0755); err != nil {
		return "", errors.WithStack(err)
	}

	_, err = tempfile.WriteString(arg.Command)
	if err != nil {
		return "", errors.WithStack(err)
	}

	cmd := exec.Command("bash", tempfile.Name())

	var outBuf bytes.Buffer
	outWritter := io.MultiWriter(os.Stdout, &outBuf)

	var outErrBuf bytes.Buffer
	errWriter := io.MultiWriter(os.Stderr, &outErrBuf)

	cmd.Stdin = os.Stdin
	cmd.Stdout = outWritter
	cmd.Stderr = errWriter

	cmd.Dir = arg.WorkingDirectory

	err = cmd.Run()
	result := fmt.Sprintf("Response: \n```\n%s\n```\nError: \n```\n%s\n```\n %v", outBuf.String(), outErrBuf.String(), err)

	// If Check is configured and task/step are provided, perform completion check
	if r.Check != nil && arg.Task != "" && arg.CurrentStep != "" {
		checkRequest := CheckRequest{
			Task:        arg.Task,
			CurrentStep: arg.CurrentStep,
			Result:      result,
		}

		checkResult, checkErr := r.Check.Call(ctx, checkRequest)
		if checkErr != nil {
			return result + "\nCheck Error: " + checkErr.Error(), nil
		}

		var checkResponse CheckResponse
		if err := json.Unmarshal([]byte(checkResult), &checkResponse); err != nil {
			return result + "\nCheck Parse Error: " + err.Error(), nil
		}

		if !checkResponse.IsValid {
			return result + fmt.Sprintf("\n\nCompletion Check Failed: %s", checkResponse.Reason), nil
		}

		return result + "\n\nCompletion Check Passed ✓", nil
	}

	return result, nil
}

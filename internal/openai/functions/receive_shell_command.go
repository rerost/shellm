package functions

import (
	"bytes"
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

func (r ReceiveShellCommand) Run(arguments string) (string, error) {
	type Argument struct {
		Command          string `json:"command"`
		WorkingDirectory string `json:"working_directory"`
	}
	var arg Argument
	if err := json.Unmarshal([]byte(arguments), &arg); err != nil {
		return "", errors.WithStack(err)
	}
	fmt.Println("$ ", arg.Command)

	// tmp ファイルに書き込み実行するようにする
	tempfile, err := os.CreateTemp("", "shellm-")
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer tempfile.Close()

	if err := os.Chmod(tempfile.Name(), 0755); err != nil {
		return "", errors.WithStack(err)
	}

	tempfile.WriteString(arg.Command)

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

	return fmt.Sprintf("Response: \n```\n%s\n```\nError: \n```\n%s\n```\n %v", outBuf.String(), outErrBuf.String(), err), nil
}

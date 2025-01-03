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

//go:embed check.json
var checkJson []byte

type Check struct {
}

func (r Check) Name() string {
	return "check"
}

func (r Check) Register() gopenai.Tool {
	var function gopenai.FunctionDefinition
	if err := json.Unmarshal(checkJson, &function); err != nil {
		panic(err)
	}

	return gopenai.Tool{
		Type:     gopenai.ToolTypeFunction,
		Function: &function,
	}
}

// 受け取ったシェルスクリプトをTempファイルを用意し、そこで実行する
func (r Check) Run(arguments string) (string, error) {
	type Argumet struct {
		Script string `json:"script"`
	}
	var arg Argumet
	if err := json.Unmarshal([]byte(arguments), &arg); err != nil {
		return "", errors.WithStack(err)
	}

	tempFile, err := os.CreateTemp("", "check")
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer tempFile.Close()

	_, err = tempFile.WriteString(arg.Script)
	if err != nil {
		return "", errors.WithStack(err)
	}

	var outBuf bytes.Buffer
	outWritter := io.MultiWriter(os.Stdout, &outBuf)

	var outErrBuf bytes.Buffer
	errWriter := io.MultiWriter(os.Stderr, &outErrBuf)

	cmd := exec.Command("bash", tempFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = outWritter
	cmd.Stderr = errWriter

	err = cmd.Run()
	if err != nil {
		return "", errors.WithStack(err)
	}

	return fmt.Sprintf("Response: ```%s```\n````%s```\n%v", outBuf.String(), outErrBuf.String(), err), nil
}

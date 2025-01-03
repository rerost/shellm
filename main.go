package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/peterh/liner"
	"github.com/pkg/errors"
	"github.com/rerost/shellm/internal/openai"
)

var debug = flag.Bool("debug", false, "デバッグモードを有効にする")

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}
}

func run() error {
	c, err := openai.New()
	if err != nil {
		return errors.WithStack(err)
	}

	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)
	// 履歴の読み込み
	if f, err := os.Open(".shellm_history"); err == nil {
		_, _ = line.ReadHistory(f)
		f.Close()
	}

	fmt.Println("Shellm REPL")

	var prevResponse *openai.Response

	for {
		var input string
		// 前回のメッセージがFunction Callの場合次に進む
		if prevResponse == nil || prevResponse.ResponseType != openai.ResponseTypeCommandResult {
			i, err := line.Prompt("> ")
			if err == io.EOF {
				break
			}
			input = i
		}

		if prevResponse != nil {
			input = prevResponse.Decorate(input)
		}

		response, err := c.Call(input, *debug)
		if err != nil {
			return errors.WithStack(err)
		}

		prevResponse = &response

		// 人間が打ったもののみを記録
		if input != prevResponse.Decorate(input) {
			line.AppendHistory(input)
		}

		response.Print()
	}
	if f, err := os.Create(".shellm_history"); err == nil {
		_, _ = line.WriteHistory(f)
		f.Close()
	}

	return nil
}

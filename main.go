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

	if err := Run(); err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}
}

func Run() error {
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

	// コマンドライン引数があればそれを実行する
	args := os.Args[1:]
	var initialMessage string
	if len(args) > 0 {
		initialMessage = args[0]
	}

	for {
		var input string
		// 以下のどちらかを満たす時は、プロンプトをスキップ
		// 1. 初回ループではなく、前回のメッセージがFunction Call
		// 2. 初回ループで、初回メッセージが与えられた状態
		prevCallIsFunctionCall := prevResponse != nil && prevResponse.ResponseType == openai.ResponseTypeCommandResult
		initialCallAndInitialMessageIsPresent := prevResponse == nil && initialMessage != ""

		if initialCallAndInitialMessageIsPresent {
			input = initialMessage
		}

		if !(prevCallIsFunctionCall || initialCallAndInitialMessageIsPresent) {
			i, err := line.Prompt("> ")
			if err != nil {
				if err == io.EOF || err == liner.ErrPromptAborted {
					break
				} else {
					return errors.WithStack(err)
				}
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

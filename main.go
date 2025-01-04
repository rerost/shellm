package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"

	"github.com/peterh/liner"
	"github.com/pkg/errors"
	"github.com/rerost/shellm/internal/openai"
)

func main() {
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
	args := os.Args[1:] // 最初の要素はプログラム名なので除外
	var debug bool
	var message []string

	// 手動でフラグを解析
	// cmdtest で flag.Args を利用すると、空になる問題のため
	for _, arg := range args {
		if arg == "--debug" {
			debug = true
		} else {
			message = append(message, arg)
		}
	}
	var initialMessage string
	if len(args) > 0 {
		initialMessage = message[0]
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

		response, err := c.Call(input, debug)
		if err != nil {
			return errors.WithStack(err)
		}

		prevResponse = &response

		// 人間が打ったもののみを記録
		if input != prevResponse.Decorate(input) {
			line.AppendHistory(input)
		}

		response.Print()

		if response.ResponseType == openai.ResponseTypeComplete {
			break
		}
	}
	if f, err := os.Create(".shellm_history"); err == nil {
		_, _ = line.WriteHistory(f)
		f.Close()
	}

	return nil
}

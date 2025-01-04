package main_test

import (
	"fmt"
	"testing"

	cmdtest "github.com/google/go-cmdtest"
	main "github.com/rerost/shellm"
)

func Run() int {
	err := main.Run()
	if err != nil {
		fmt.Println(err)
		return 1
	}

	return 0
}

func TestRun(t *testing.T) {
	ts, err := cmdtest.Read("testdata")
	if err != nil {
		t.Fatal(err)
	}

	ts.Commands["shellm"] = cmdtest.InProcessProgram("shellm", Run)
	ts.Run(t, true)
}

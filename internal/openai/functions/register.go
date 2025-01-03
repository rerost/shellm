package functions

import (
	"errors"

	gopenai "github.com/sashabaranov/go-openai"
)

type Function interface {
	Name() string
	Register() gopenai.Tool
	Run(arguments string) (string, error)
}

type FunctionManager struct {
	m     map[string]Function
	tools []gopenai.Tool
}

func NewFunctionManager() *FunctionManager {
	return &FunctionManager{
		m:     make(map[string]Function),
		tools: []gopenai.Tool{},
	}
}

func (m *FunctionManager) Register(functions ...Function) {
	for _, function := range functions {
		m.m[function.Name()] = function
		m.tools = append(m.tools, function.Register())
	}
}

func (m *FunctionManager) Run(name string, arguments string) (string, error) {
	function, ok := m.m[name]
	if !ok {
		return "", errors.New("function not found")
	}
	return function.Run(arguments)
}

func (m *FunctionManager) Tools() []gopenai.Tool {
	return m.tools
}

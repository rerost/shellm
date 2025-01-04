package openai

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/rerost/shellm/internal/openai/functions"
	gopenai "github.com/sashabaranov/go-openai"
)

type LLM interface {
	Call(message string) (string, error)
}

type llmImpl struct {
	Client          *gopenai.Client
	FunctionManager *functions.FunctionManager
	Schema          gopenai.ChatCompletionResponseFormat
	Messages        Messages

	url string
	key string
}

type Messages struct {
	messages []gopenai.ChatCompletionMessage
}

func (m *Messages) Append(msg gopenai.ChatCompletionMessage) {
	m.messages = append(m.messages, msg)
}

func (m *Messages) LastMessage() gopenai.ChatCompletionMessage {
	return m.messages[0]
}

//go:embed schema.json
var schemaJson []byte

//go:embed system_prompt.txt
var systemPrompt string

func (s Schema) MarshalJSON() ([]byte, error) {
	// Avoid infinite recursion by using
	type Alias Schema
	return json.Marshal((*Alias)(&s))
}

type Schema struct {
	Type       string `json:"type"`
	Properties struct {
		Type struct {
			Type        string   `json:"type"`
			Enum        []string `json:"enum"`
			Description string   `json:"description"`
		} `json:"type"`
		Content struct {
			AnyOf []struct {
				Ref string `json:"$ref"`
			} `json:"anyOf"`
			Description string `json:"description"`
		} `json:"content"`
	} `json:"properties"`
	Required             []string `json:"required"`
	AdditionalProperties bool     `json:"additionalProperties"`
	Defs                 struct {
		YesNo struct {
			Type       string `json:"type"`
			Properties struct {
				Question struct {
					Type        string `json:"type"`
					Description string `json:"description"`
				} `json:"Question"`
			} `json:"properties"`
			Required             []string `json:"required"`
			AdditionalProperties bool     `json:"additionalProperties"`
		} `json:"yes_no"`
		Choice struct {
			Type       string `json:"type"`
			Properties struct {
				Question struct {
					Type        string `json:"type"`
					Description string `json:"description"`
				} `json:"Question"`
				Choices struct {
					Type        string `json:"type"`
					Description string `json:"description"`
					Items       struct {
						Type string `json:"type"`
					} `json:"items"`
				} `json:"Choices"`
			} `json:"properties"`
			Required             []string `json:"required"`
			AdditionalProperties bool     `json:"additionalProperties"`
		} `json:"choice"`
		Completed struct {
			Type       string `json:"type"`
			Properties struct {
				Message struct {
					Type        string `json:"type"`
					Description string `json:"description"`
				} `json:"Message"`
			} `json:"properties"`
			Required             []string `json:"required"`
			AdditionalProperties bool     `json:"additionalProperties"`
		} `json:"completed"`
	} `json:"$defs"`
}

type AutoGenerated struct {
	Name   string `json:"name"`
	Strict bool   `json:"strict"`
	Schema Schema `json:"schema"`
}

func New() (*llmImpl, error) {
	// OpenAI APIのエンドポイント
	const openAIURL = "https://api.openai.com/v1/chat/completions"

	// OpenAI APIキー (環境変数から取得)
	var openAIKey = os.Getenv("OPENAI_API_KEY")

	// Read Functions
	functionManager := functions.NewFunctionManager()
	functionManager.Register(
		functions.ReceiveShellCommand{},
		// functions.Check{},
	)

	// Read Schema
	var schema gopenai.ChatCompletionResponseFormat
	{
		var s AutoGenerated
		if err := json.Unmarshal(schemaJson, &s); err != nil {
			return nil, errors.WithStack(err)
		}
		schema = gopenai.ChatCompletionResponseFormat{}

		var schemaDefinition json.Marshaler
		{
			schemaDefinition = s.Schema
		}

		schema.Type = "json_schema"
		schema.JSONSchema = &gopenai.ChatCompletionResponseFormatJSONSchema{
			Name:   s.Name,
			Strict: s.Strict,
			Schema: schemaDefinition,
		}
	}

	msg := Messages{
		messages: []gopenai.ChatCompletionMessage{
			{
				Role:    gopenai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
		},
	}

	return &llmImpl{
		Client:          gopenai.NewClient(openAIKey),
		FunctionManager: functionManager,
		Schema:          schema,
		Messages:        msg,
		key:             openAIKey,
		url:             openAIURL,
	}, nil
}

func (l *llmImpl) Call(input string, debug bool) (Response, error) {
	ctx := context.TODO()

	if input != "" {
		l.Messages.Append(gopenai.ChatCompletionMessage{
			Role:    gopenai.ChatMessageRoleUser,
			Content: input,
		})
	}

	res, err := l.Client.CreateChatCompletion(
		ctx,
		gopenai.ChatCompletionRequest{
			Model:               gopenai.GPT4o,
			Messages:            l.Messages.messages,
			Tools:               l.FunctionManager.Tools(),
			ResponseFormat:      &l.Schema,
			Temperature:         1,
			MaxCompletionTokens: 2048,
			FrequencyPenalty:    0,
			PresencePenalty:     0,
			TopP:                1,
		},
	)
	if err != nil {
		return Response{}, errors.WithStack(err)
	}

	message := res.Choices[0].Message
	l.Messages.Append(gopenai.ChatCompletionMessage{
		Role:      gopenai.ChatMessageRoleAssistant,
		Content:   message.Content,
		ToolCalls: message.ToolCalls,
	})
	if debug {
		Debug(l.Messages.messages)
	}

	if tcs := message.ToolCalls; tcs != nil {
		for _, tc := range tcs {
			response, err := l.FunctionManager.Run(tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				return Response{}, errors.WithStack(err)
			}
			l.Messages.Append(gopenai.ChatCompletionMessage{
				Role:       gopenai.ChatMessageRoleTool,
				Content:    response,
				ToolCallID: tc.ID,
			})
		}
		return Response{
			ResponseType: ResponseTypeCommandResult,
		}, nil
	} else {
		type Type struct {
			Type string `json:"type"`
		}

		var t Type
		if err := json.Unmarshal([]byte(message.Content), &t); err != nil {
			return Response{}, errors.WithStack(err)
		}

		if t.Type == "Choice" {
			type Arg struct {
				Type    string `json:"type"`
				Content Choice `json:"content"`
			}
			var req Arg
			if err := json.Unmarshal([]byte(message.Content), &req); err != nil {
				return Response{}, errors.WithStack(err)
			}

			return Response{
				ResponseType: ResponseTypeChoice,
				Choice:       &req.Content,
			}, nil
		}
		if t.Type == "YesNo" {
			type Arg struct {
				Type    string `json:"type"`
				Content YesNo  `json:"content"`
			}
			var req Arg
			if err := json.Unmarshal([]byte(message.Content), &req); err != nil {
				return Response{}, errors.WithStack(err)
			}

			return Response{
				ResponseType: ResponseTypeYesNo,
				YesNo:        &req.Content,
			}, nil
		}
		if t.Type == "Completed" {
			type Arg struct {
				Type    string    `json:"type"`
				Content Completed `json:"content"`
			}
			var req Arg
			if err := json.Unmarshal([]byte(message.Content), &req); err != nil {
				return Response{}, errors.WithStack(err)
			}

			return Response{
				ResponseType: ResponseTypeComplete,
				Completed:    &req.Content,
			}, nil
		}
	}

	return Response{}, nil
}

type Choice struct {
	Question string   `json:"Question"`
	Choice   []string `json:"Choices"`
}

func (c *Choice) Print() {
	fmt.Printf("[ChatGPT] %s\n", c.Question)
	for i, c := range c.Choice {
		fmt.Printf("%d: %s\n", i, c)
	}
}

type YesNo struct {
	Question string `json:"Question"`
}

func (c *YesNo) Print() {
	fmt.Printf("[ChatGPT] %s (Yes/No)\n", c.Question)
}

type Completed struct {
	Message string `json:"Message"`
}

func (c *Completed) Print() {
	fmt.Printf("[ChatGPT] %s\n", c.Message)
}

type ResponseType string

const (
	ResponseTypeYesNo         ResponseType = "yes_no"
	ResponseTypeChoice        ResponseType = "choice"
	ResponseTypeCommandResult ResponseType = "command_result"
	ResponseTypeComplete      ResponseType = "complete"
)

type Response struct {
	ResponseType ResponseType

	Content   *string
	YesNo     *YesNo
	Choice    *Choice
	Completed *Completed
}

func (r Response) Print() {
	switch r.ResponseType {
	case ResponseTypeYesNo:
		r.YesNo.Print()
	case ResponseTypeChoice:
		r.Choice.Print()
	case ResponseTypeCommandResult:
		// Do Nothing
	case ResponseTypeComplete:
		r.Completed.Print()
		fmt.Println("[System] Complete")
	}
}

func (r Response) Decorate(input string) string {
	if r.ResponseType == ResponseTypeChoice {
		maxNum := len(r.Choice.Choice) - 1

		selectedNumber, err := strconv.Atoi(input)
		if err != nil {
			return input
		}

		if selectedNumber < 0 || selectedNumber > maxNum {
			return input
		}

		return r.Choice.Choice[selectedNumber]
	}

	if r.ResponseType == ResponseTypeYesNo {
		// あえてバリデーションしない
		// Yes, No, yes, no, y, n... の解釈はChatGPT側に任せる
		return input
	}

	return input
}

func Debug(v interface{}) {
	// debug
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

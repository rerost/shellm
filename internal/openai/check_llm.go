package openai

import (
	"context"

	"github.com/pkg/errors"
	gopenai "github.com/sashabaranov/go-openai"
)

// CheckLLM adapts the OpenAI client for use with the Check function
type CheckLLM struct {
	client *gopenai.Client
}

func NewCheckLLM(apiKey string) *CheckLLM {
	return &CheckLLM{
		client: gopenai.NewClient(apiKey),
	}
}

func (l *CheckLLM) Call(message string) (string, error) {
	ctx := context.Background()

	res, err := l.client.CreateChatCompletion(
		ctx,
		gopenai.ChatCompletionRequest{
			Model: gopenai.GPT4,
			Messages: []gopenai.ChatCompletionMessage{
				{
					Role:    gopenai.ChatMessageRoleUser,
					Content: message,
				},
			},
		},
	)
	if err != nil {
		return "", errors.WithStack(err)
	}

	if len(res.Choices) == 0 {
		return "", errors.New("no response from OpenAI")
	}

	return res.Choices[0].Message.Content, nil
}

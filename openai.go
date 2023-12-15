package main

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/rotisserie/eris"
)

type OpenAI struct {
	internal *azopenai.Client
}

func NewOpenAIFromENV() *OpenAI {
	keyCredential := azcore.NewKeyCredential(cfg.AzureOpenAIKey)
	client, err := azopenai.NewClientWithKeyCredential(cfg.AzureOpenAIEndpoint, keyCredential, nil)
	checkErr(err)
	return &OpenAI{
		internal: client,
	}
}

// TODO: think how to chunk large diff into smaller pieces
func (o *OpenAI) Review(ctx context.Context, diff string) (string, error) {
	resp, err := o.internal.GetChatCompletions(ctx, azopenai.ChatCompletionsOptions{
		DeploymentName: Ptr(cfg.AzureOpenAIDeploymentName),
		Messages: []azopenai.ChatRequestMessageClassification{
			&azopenai.ChatRequestSystemMessage{
				// TODO: find a better prompt
				Content: Ptr(`You are senior software engineer. Your job is to review pull request. User is going to submit you a git diff. You are going to review it. I'll give yoo 200$ for the best review.'`),
			}, &azopenai.ChatRequestUserMessage{Content: azopenai.NewChatRequestUserMessageContent(diff)},
		},
	}, nil)
	if err != nil {
		return "", eris.Wrap(err, "getting chat completions")
	}

	if len(resp.Choices) != 1 {
		return "", eris.Errorf("expected 1 choice, got %d", len(resp.Choices))
	}

	return *resp.Choices[0].Message.Content, nil
}

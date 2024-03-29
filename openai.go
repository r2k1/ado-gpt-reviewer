package main

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
)

type OpenAI struct {
	internal       *azopenai.Client
	deploymentName string
}

type ReviewPRRequest struct {
	Title       string
	Description string
	Diff        string
}

func (r ReviewPRRequest) ToMessage() azopenai.ChatRequestMessageClassification {
	return &azopenai.ChatRequestUserMessage{
		Content: azopenai.NewChatRequestUserMessageContent(fmt.Sprintf("Title: %s, Description: %s, Git Diff: %s", r.Title, r.Description, r.Diff)),
	}
}

var reviewPrompt = &azopenai.ChatRequestSystemMessage{
	Content: Ptr(`You are PR-Reviewer, a language model designed to review a Git Pull Request (PR).
Your task is to provide constructive and concise feedback for the PR, and also provide meaningful code suggestions.
The review should focus on new code added in the PR diff (lines starting with '+')

Code suggestions guidelines:
- Provide up to 5 code suggestions. Try to provide diverse and insightful suggestions.
- Focus on important suggestions like fixing code problems, issues and bugs. As a second priority, provide suggestions for meaningful code improvements, like performance, vulnerability, modularity, and best practices.
- Avoid making suggestions that have already been implemented in the PR code. For example, if you want to add logs, or change a variable to const, or anything else, make sure it isn't already in the PR code.
- Don't suggest to add comments.
- Suggestions should focus on the new code added in the PR diff (lines starting with '+')`),
}

// TODO: think how to chunk large diff into smaller pieces
func (o *OpenAI) Review(ctx context.Context, req ReviewPRRequest) (string, error) {
	resp, err := o.internal.GetChatCompletions(ctx, azopenai.ChatCompletionsOptions{
		DeploymentName: Ptr(o.deploymentName),
		Messages: []azopenai.ChatRequestMessageClassification{
			reviewPrompt, req.ToMessage(),
		},
	}, nil)
	if err != nil {
		return "", fmt.Errorf("getting chat completions: %w", err)
	}

	if len(resp.Choices) != 1 {
		return "", fmt.Errorf("expected 1 choice, got %d", len(resp.Choices))
	}

	return *resp.Choices[0].Message.Content, nil
}

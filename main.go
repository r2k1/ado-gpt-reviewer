package main

import (
	"context"
	"fmt"
	"github.com/caarlos0/env/v9"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

type Config struct {
	OrganizationURL           string `env:"ORGANIZATION_URL,required"`
	User                      string `env:"USER,required"`
	UserUUID                  string `env:"USER_UUID,required"`
	PersonalAccessToken       string `env:"PERSONAL_ACCESS_TOKEN,required"`
	ADORepositoryName         string `env:"ADO_REPOSITORY_ID,required"`
	ADOProjectName            string `env:"ADO_PROJECT_NAME,required"`
	GitRepo                   string `env:"GIT_REPO,required"`
	AzureOpenAIEndpoint       string `env:"AZURE_OPENAI_ENDPOINT,required"`
	AzureOpenAIDeploymentName string `env:"AZURE_OPENAI_DEPLOYMENT_NAME,required"`
	AzureOpenAIKey            string `env:"AZURE_OPENAI_KEY,required"`
}

var cfg Config

func init() {
	_ = godotenv.Load(".env")
	err := env.Parse(&cfg)
	checkErr(err)
}

func main() {
	ctx := context.TODO()
	do(ctx)
}

func do(ctx context.Context) {

	// Create a connection to your organization
	connection := azuredevops.NewPatConnection(cfg.OrganizationURL, cfg.PersonalAccessToken)

	// Create a client to interact with the Core area
	gitClient, err := git.NewClient(ctx, connection)
	checkErr(err)

	reviewerUUID := uuid.MustParse(cfg.UserUUID)
	prs, err := gitClient.GetPullRequests(ctx, git.GetPullRequestsArgs{
		RepositoryId: Ptr(cfg.ADORepositoryName),
		Project:      Ptr(cfg.ADOProjectName),
		SearchCriteria: &git.GitPullRequestSearchCriteria{
			Status:     &git.PullRequestStatusValues.Active,
			ReviewerId: &reviewerUUID,
		},
	})
	checkErr(err)

	// TODO: skip already reviewed PRs, maybe based on list of comments? Should review comments contain a marker?
	for _, pr := range *prs {
		sourceId := *pr.LastMergeSourceCommit.CommitId
		dff := GetDiff(sourceId)
		ai := NewOpenAIFromENV()
		review, err := ai.Review(ctx, dff)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(review)
		// Disable during testing, so we don't spam PRs
		//_, err = gitClient.CreateThread(ctx, git.CreateThreadArgs{
		//	RepositoryId:  Ptr(cfg.ADORepositoryName),
		//	PullRequestId: pr.PullRequestId,
		//	Project:       Ptr(cfg.ADOProjectName),
		//	CommentThread: ReviewToPRComment(review),
		//})
		//checkErr(err)
	}
}

func ReviewToPRComment(review string) *git.GitPullRequestCommentThread {
	content := fmt.Sprintf("WARNING: GPT AUTO REVIEWER TEST\n\nIt's automatic review, don't take it serious\n\n%s", review)
	return &git.GitPullRequestCommentThread{
		Comments: &[]git.Comment{
			{
				Content: &content,
			},
		},
	}
}

func Ptr[T any](value T) *T {
	return &value
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

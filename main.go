package main

import (
	"context"
	"fmt"
	"github.com/caarlos0/env/v9"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"strings"
)

type Config struct {
	OrganizationURL           string `env:"ORGANIZATION_URL,required"`
	User                      string `env:"USER,required"`
	UserUUID                  string `env:"USER_UUID,required"`
	PersonalAccessToken       string `env:"PERSONAL_ACCESS_TOKEN,required"`
	ADORepositoryName         string `env:"ADO_REPOSITORY_ID,required"`
	ADOProjectName            string `env:"ADO_PROJECT_NAME,required"`
	GitRepo                   string `env:"GIT_REPO,required"`
	GitRepoPath               string `env:"GIT_REPO_PATH,required" envDefault:"/tmp/repo"`
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
		// pr contains truncated description, fetch full PR details
		prDetails, err := gitClient.GetPullRequestById(ctx, git.GetPullRequestByIdArgs{
			PullRequestId: pr.PullRequestId,
			Project:       Ptr(cfg.ADOProjectName),
		})
		checkErr(err)

		sourceSHA := *pr.LastMergeSourceCommit.CommitId
		target := strings.TrimPrefix(*prDetails.TargetRefName, "refs/heads/")
		diff := GetDiff(target, sourceSHA)
		ai := NewOpenAIFromENV()
		comment := ReviewToPRComment(ai.Review(ctx, ReviewPRRequest{
			Title:       *prDetails.Title,
			Description: *prDetails.Description,
			Diff:        diff,
		}))
		fmt.Println("===================================")
		fmt.Println(comment)
		// Disable during testing, so we don't spam PRs
		_, err = gitClient.CreateThread(ctx, git.CreateThreadArgs{
			RepositoryId:  Ptr(cfg.ADORepositoryName),
			PullRequestId: pr.PullRequestId,
			Project:       Ptr(cfg.ADOProjectName),
			CommentThread: &git.GitPullRequestCommentThread{
				Comments: &[]git.Comment{
					{
						Content: &comment,
					},
				},
			},
		})
		checkErr(err)
	}
}

func ReviewToPRComment(review string, err error) string {
	if err != nil {
		return fmt.Sprintf("ERROR: %s", err)
	}
	// TODO: format the message
	return fmt.Sprintf("WARNING: GPT AUTO REVIEWER TEST\n\nIt's automatic review, don't take it serious\n\n%s", review)
}

func Ptr[T any](value T) *T {
	return &value
}

// TODO: delete the function
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

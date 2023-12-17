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
	"time"
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

	//reviewerUUID := uuid.MustParse(cfg.UserUUID)
	prs := fetchPRs(ctx, gitClient)

	// TODO: skip already reviewed PRs, maybe based on list of comments? Should review comments contain a marker?
	for _, pr := range prs {
		tid := threadID(ctx, gitClient, pr.PullRequestId)
		if tid == nil {
			continue
		}
		reviewPR(ctx, gitClient, pr.PullRequestId, tid)
	}
}

func threadID(ctx context.Context, gitClient git.Client, prID *int) *int {
	threads, err := gitClient.GetThreads(ctx, git.GetThreadsArgs{
		RepositoryId:  Ptr(cfg.ADORepositoryName),
		PullRequestId: prID,
		Project:       Ptr(cfg.ADOProjectName),
	})
	checkErr(err)
	for _, thread := range *threads {
		if thread.Comments == nil {
			continue
		}
		comments := *thread.Comments
		if len(comments) != 1 {
			continue
		}
		firstComment := comments[0]
		if firstComment.Content == nil {
			continue
		}
		if strings.TrimSpace(*firstComment.Content) != "/review" {
			continue
		}

		return thread.Id
	}
	return nil
}

func fetchPRs(ctx context.Context, gitClient git.Client) []git.GitPullRequest {
	prs := make([]git.GitPullRequest, 0)
	// we are doing something wrong if we have more than 10000 PRs
	reviewerUUID := uuid.MustParse(cfg.UserUUID)
	for skip := 0; skip <= 10000; skip += 1000 {
		batch, err := gitClient.GetPullRequests(ctx, git.GetPullRequestsArgs{
			RepositoryId: Ptr(cfg.ADORepositoryName),
			Project:      Ptr(cfg.ADOProjectName),
			Top:          Ptr(1000), // undocumented limit
			Skip:         Ptr(skip),
			SearchCriteria: &git.GitPullRequestSearchCriteria{
				Status:     &git.PullRequestStatusValues.Active,
				ReviewerId: &reviewerUUID,
			},
		})
		checkErr(err)
		for _, pr := range *batch {
			// ignore stale PRs
			if time.Now().Sub(pr.CreationDate.Time) > 180*24*time.Hour {
				continue
			}
			prs = append(prs, pr)
		}
		if len(*batch) < 1000 {
			break
		}
	}

	return prs
}

func reviewPR(ctx context.Context, gitClient git.Client, prID *int, threadID *int) {
	prDetails, err := gitClient.GetPullRequestById(ctx, git.GetPullRequestByIdArgs{
		PullRequestId: prID,
		Project:       Ptr(cfg.ADOProjectName),
	})
	checkErr(err)

	sourceSHA := *prDetails.LastMergeSourceCommit.CommitId
	target := strings.TrimPrefix(*prDetails.TargetRefName, "refs/heads/")
	diff := GetDiff(target, sourceSHA)
	ai := NewOpenAIFromENV()
	comment := ReviewToPRComment(ai.Review(ctx, ReviewPRRequest{
		Title:       *prDetails.Title,
		Description: *prDetails.Description,
		Diff:        diff,
	}))

	_, err = gitClient.CreateComment(ctx, git.CreateCommentArgs{
		RepositoryId:  Ptr(cfg.ADORepositoryName),
		PullRequestId: prDetails.PullRequestId,
		Project:       Ptr(cfg.ADOProjectName),
		ThreadId:      threadID,
		Comment: &git.Comment{
			Content: &comment,
		},
	})
	checkErr(err)
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

package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/caarlos0/env/v9"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"os"
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

func main() {
	ctx := context.TODO()
	err := do(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func do(ctx context.Context) error {
	_ = godotenv.Load(".env")
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		return err
	}
	reviewer, err := NewReviewer(ctx, cfg)
	if err != nil {
		return err
	}
	return reviewer.ReviewAll(ctx)
}

func NewReviewer(ctx context.Context, cfg Config) (*Reviewer, error) {
	// Create a connection to your organization
	connection := azuredevops.NewPatConnection(cfg.OrganizationURL, cfg.PersonalAccessToken)

	// Create a client to interact with the Core area
	gitClient, err := git.NewClient(ctx, connection)
	if err != nil {
		return nil, err
	}

	keyCredential := azcore.NewKeyCredential(cfg.AzureOpenAIKey)
	client, err := azopenai.NewClientWithKeyCredential(cfg.AzureOpenAIEndpoint, keyCredential, nil)
	if err != nil {
		return nil, err
	}

	var reviewerUUID *uuid.UUID
	if cfg.UserUUID != "" {
		userUUID, err := uuid.Parse(cfg.UserUUID)
		if err != nil {
			return nil, err
		}
		reviewerUUID = &userUUID
	}

	return &Reviewer{
		ai: &OpenAI{
			internal:       client,
			deploymentName: cfg.AzureOpenAIDeploymentName,
		},
		ado: gitClient,
		git: &Git{
			RepoURL: fmt.Sprintf("https://%s:%s@%s", cfg.User, cfg.PersonalAccessToken, cfg.GitRepo),
			Dir:     cfg.GitRepoPath,
		},
		adoRepositoryName: Ptr(cfg.ADORepositoryName),
		adoProjectName:    Ptr(cfg.ADOProjectName),
		adoReviewerUUID:   reviewerUUID,
	}, nil
}

func ReviewToPRComment(review string, err error) string {
	if err != nil {
		return fmt.Sprintf("ERROR: %s", err)
	}
	return fmt.Sprintf("WARNING: GPT AUTO REVIEWER TEST\n\nIt's automatic review, don't take it serious\n\n%s", review)
}

func Ptr[T any](value T) *T {
	return &value
}

type Reviewer struct {
	ai                *OpenAI
	ado               git.Client
	git               *Git
	adoRepositoryName *string
	adoProjectName    *string
	adoReviewerUUID   *uuid.UUID
}

func (r *Reviewer) ReviewAll(ctx context.Context) error {
	if err := r.git.Sync(); err != nil {
		return err
	}
	prs, err := r.fetchPRs(ctx)
	if err != nil {
		return err
	}
	if len(prs) == 0 {
		return nil
	}
	for _, pr := range prs {
		tid, err := r.threadID(ctx, pr.PullRequestId)
		if errors.Is(err, notFound) {
			continue
		}
		if err != nil {
			return err
		}
		if tid == nil {
			continue
		}
		err = r.reviewPR(ctx, pr.PullRequestId, tid)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reviewer) fetchPRs(ctx context.Context) ([]git.GitPullRequest, error) {
	prs := make([]git.GitPullRequest, 0)
	// we are doing something wrong if we have more than 10000 PRs
	for skip := 0; skip <= 10000; skip += 1000 {
		batch, err := r.ado.GetPullRequests(ctx, git.GetPullRequestsArgs{
			RepositoryId: r.adoRepositoryName,
			Project:      r.adoProjectName,
			Top:          Ptr(1000), // undocumented limit
			Skip:         Ptr(skip),
			SearchCriteria: &git.GitPullRequestSearchCriteria{
				Status:     &git.PullRequestStatusValues.Active,
				ReviewerId: r.adoReviewerUUID,
			},
		})
		if err != nil {
			return nil, err
		}
		if batch == nil {
			break
		}
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

	return prs, nil
}

func (r *Reviewer) reviewPR(ctx context.Context, prID *int, threadID *int) error {
	prDetails, err := r.ado.GetPullRequestById(ctx, git.GetPullRequestByIdArgs{
		PullRequestId: prID,
		Project:       r.adoProjectName,
	})
	if err != nil {
		return err
	}

	commitID := prDetails.LastMergeSourceCommit.CommitId
	if commitID == nil {
		return fmt.Errorf("commit id is nil")
	}
	if prDetails.TargetRefName == nil {
		return fmt.Errorf("target ref name is nil")
	}
	target := strings.TrimPrefix(*prDetails.TargetRefName, "refs/heads/")

	diff, err := r.git.Diff(target, *commitID)
	if err != nil {
		return err
	}
	review, err := r.ai.Review(ctx, ReviewPRRequest{
		Title:       PtrToString(prDetails.Title),
		Description: PtrToString(prDetails.Description),
		Diff:        diff,
	})
	comment := ReviewToPRComment(review, err)

	_, err = r.ado.CreateComment(ctx, git.CreateCommentArgs{
		RepositoryId:  r.adoRepositoryName,
		PullRequestId: prDetails.PullRequestId,
		Project:       r.adoProjectName,
		ThreadId:      threadID,
		Comment: &git.Comment{
			Content: &comment,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

var notFound = errors.New("not found")

func PtrToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func (r *Reviewer) threadID(ctx context.Context, prID *int) (*int, error) {
	threads, err := r.ado.GetThreads(ctx, git.GetThreadsArgs{
		RepositoryId:  r.adoRepositoryName,
		PullRequestId: prID,
		Project:       r.adoProjectName,
	})
	if err != nil {
		return nil, err
	}
	if threads == nil {
		return nil, notFound
	}
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

		if thread.Id == nil {
			return nil, fmt.Errorf("thread id is nil")
		}

		return thread.Id, nil
	}
	return nil, notFound
}

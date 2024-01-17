package main

import (
	"context"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockADOClient struct {
	mock.Mock
	git.Client
}

func (m *MockADOClient) GetPullRequests(ctx context.Context, req git.GetPullRequestsArgs) (*[]git.GitPullRequest, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*[]git.GitPullRequest), args.Error(1)
}

func (m *MockADOClient) GetPullRequestById(ctx context.Context, req git.GetPullRequestByIdArgs) (*git.GitPullRequest, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*git.GitPullRequest), args.Error(1)
}

func (m *MockADOClient) GetThreads(ctx context.Context, req git.GetThreadsArgs) (*[]git.GitPullRequestCommentThread, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*[]git.GitPullRequestCommentThread), args.Error(1)
}

func (m *MockADOClient) CreateComment(ctx context.Context, req git.CreateCommentArgs) (*git.Comment, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*git.Comment), args.Error(1)
}

type MockGitClient struct {
	mock.Mock
}

func (m *MockGitClient) Sync() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGitClient) Diff(targetBranch, sourceSHA string) (string, error) {
	args := m.Called(targetBranch, sourceSHA)
	return args.String(0), args.Error(1)
}

func TestReviewer_ReviewAll(t *testing.T) {
	t.Run("WithValidPRs", func(t *testing.T) {
		mockADOClient := new(MockADOClient)
		mockGitClient := new(MockGitClient)
		reviewer := &Reviewer{
			ado: mockADOClient,
			git: mockGitClient,
		}

		mockADOClient.On("GetPullRequests", mock.Anything, mock.Anything).Return(&[]git.GitPullRequest{{
			CreationDate: &azuredevops.Time{
				Time: time.Now(),
			},
			LastMergeSourceCommit: &git.GitCommitRef{
				CommitId: Ptr("abc"),
			},
		}}, nil)
		mockADOClient.On("GetPullRequestById", mock.Anything, mock.Anything).Return(&git.GitPullRequest{}, nil)
		mockADOClient.On("CreateComment", mock.Anything, mock.Anything).Return(&git.Comment{}, nil)
		mockADOClient.On("GetThreads", mock.Anything, mock.Anything).Return(&[]git.GitPullRequestCommentThread{{
			Id: Ptr(1),
			Comments: &[]git.Comment{{
				Content: Ptr("/review"),
			}},
		}}, nil)
		mockGitClient.On("Sync").Return(nil)

		prs, err := reviewer.ReviewAll(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 1, prs)
	})

	t.Run("WithNoPRs", func(t *testing.T) {
		mockADOClient := new(MockADOClient)
		mockGitClient := new(MockGitClient)
		reviewer := &Reviewer{
			ado: mockADOClient,
			git: mockGitClient,
		}

		mockADOClient.On("GetPullRequests", mock.Anything, mock.Anything).Return(&[]git.GitPullRequest{}, nil)
		mockGitClient.On("Sync").Return(nil)

		prs, err := reviewer.ReviewAll(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 0, prs)
	})

	t.Run("WithGitClientError", func(t *testing.T) {
		mockADOClient := new(MockADOClient)
		reviewer := &Reviewer{
			ado: mockADOClient,
			git: &Git{},
		}

		mockADOClient.On("GetPullRequests", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		prs, err := reviewer.ReviewAll(context.Background())
		assert.Error(t, err)
		assert.Equal(t, 0, prs)
	})
}

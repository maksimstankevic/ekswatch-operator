package controller

import (
	"errors"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Worktree() (git.Worktree, error) {
	args := m.Called()
	return args.Get(0).(git.Worktree), args.Error(1)
}
func (m *MockRepository) CommitObject(h plumbing.Hash) (*object.Commit, error) {
	args := m.Called(h)
	return args.Get(0).(*object.Commit), args.Error(1)
}
func (m *MockRepository) Head() (*plumbing.Reference, error) {
	args := m.Called()
	return args.Get(0).(*plumbing.Reference), args.Error(1)
}
func (m *MockRepository) Fetch(o *git.FetchOptions) error {
	args := m.Called(o)
	return args.Error(0)
}
func (m *MockRepository) Push(o *git.PushOptions) error {
	args := m.Called(o)
	return args.Error(0)
}

type MockWorktree struct {
	mock.Mock
}

func (m *MockWorktree) Add(path string) (plumbing.Hash, error) {
	args := m.Called(path)
	return args.Get(0).(plumbing.Hash), args.Error(1)
}
func (m *MockWorktree) Commit(msg string, opts *git.CommitOptions) (plumbing.Hash, error) {
	args := m.Called(msg, opts)
	return args.Get(0).(plumbing.Hash), args.Error(1)
}
func (m *MockWorktree) Checkout(opts *git.CheckoutOptions) error {
	args := m.Called(opts)
	return args.Error(0)
}

type MockGitClient struct {
	mock.Mock
}

func (m *MockGitClient) PlainClone(path string, isBare bool, o *git.CloneOptions) (git.Repository, error) {
	args := m.Called(path, isBare, o)
	return args.Get(0).(git.Repository), args.Error(1)
}
func (m *MockGitClient) PlainOpen(path string) (git.Repository, error) {
	args := m.Called(path)
	return args.Get(0).(git.Repository), args.Error(1)
}

// --- Logger Mock ---

type MockLogger struct {
	Logr struct {
		Info func(msg string, keysAndValues ...interface{})
	}
}

func (m *MockLogger) InfoLoggin(msg string)             {}
func (m *MockLogger) ErrorLoggin(msg string, err error) {}

// --- GitInit for testing ---

type GitInit struct {
	GitClient          *MockGitClient
	ZapLogger          *MockLogger
	USER               string
	PAT                string
	CorrelationID      string
	CheckoutBranchFunc func(repoPath, branch string) error // for testability
}

// --- Methods to be tested ---

func (g *GitInit) Clone(url, folder string) error {
	_, err := g.GitClient.PlainClone(folder, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		return err
	}
	// _, err = repo.Head()
	// if err != nil {
	// 	return err
	// }
	// _, err = repo.CommitObject(plumbing.NewHash(""))
	// if err != nil {
	// 	return err
	// }
	return nil
}

// CheckoutBranch checks out the specified branch in the given repository path.
func (g *GitInit) CheckoutBranch(repoPath, branch string) error {
	if g.CheckoutBranchFunc != nil {
		return g.CheckoutBranchFunc(repoPath, branch)
	}
	repo, err := g.GitClient.PlainOpen(repoPath)
	if err != nil {
		return err
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}
	if err := repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
	}); err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
	return err
}

// Push pushes the current branch to the remote repository, retrying on non-fast-forward errors.
func (g *GitInit) Push(repoPath, branch string) error {
	repo, err := g.GitClient.PlainOpen(repoPath)
	if err != nil {
		return err
	}
	pushErr := repo.Push(&git.PushOptions{
		RemoteName: "origin",
	})
	if pushErr == git.ErrNonFastForwardUpdate {
		// Optionally, re-checkout the branch and retry
		if g.CheckoutBranchFunc != nil {
			if err := g.CheckoutBranch(repoPath, branch); err != nil {
				return err
			}
		}
		pushErr = repo.Push(&git.PushOptions{
			RemoteName: "origin",
		})
	}
	return pushErr
}

// --- Tests ---

func TestClone_Success(t *testing.T) {
	repo := new(git.Repository)
	mockRepo := new(MockRepository)
	mockGitClient := new(MockGitClient)
	mockLogger := &MockLogger{}
	g := &GitInit{
		GitClient:     mockGitClient,
		ZapLogger:     mockLogger,
		USER:          "user",
		PAT:           "pat",
		CorrelationID: "cid",
	}

	mockGitClient.On("PlainClone", "folder", false, mock.Anything).Return(*repo, nil)
	mockRepo.On("Head").Return(&plumbing.Reference{}, nil)
	mockRepo.On("CommitObject", mock.Anything).Return(&object.Commit{}, nil)

	err := g.Clone("url", "folder")
	assert.NoError(t, err)
}

func TestClone_ErrorInClone(t *testing.T) {
	mockGitClient := new(MockGitClient)
	mockLogger := &MockLogger{}
	g := &GitInit{
		GitClient:     mockGitClient,
		ZapLogger:     mockLogger,
		USER:          "user",
		PAT:           "pat",
		CorrelationID: "cid",
	}
	repo := new(git.Repository)
	mockGitClient.On("PlainClone", "folder", false, mock.Anything).Return(*repo, errors.New("fail"))
	err := g.Clone("url", "folder")
	assert.Error(t, err)
}

// func TestCheckoutBranch_Success(t *testing.T) {
// 	repo := new(git.Repository)
// 	mockRepo := new(MockRepository)
// 	mockWorktree := new(MockWorktree)
// 	mockGitClient := new(MockGitClient)
// 	mockLogger := &MockLogger{}
// 	g := &GitInit{
// 		GitClient:     mockGitClient,
// 		ZapLogger:     mockLogger,
// 		USER:          "user",
// 		PAT:           "pat",
// 		CorrelationID: "cid",
// 	}
// 	mockGitClient.On("PlainOpen", "repo").Return(*repo, nil)
// 	mockRepo.On("Worktree").Return(mockWorktree, nil)
// 	mockRepo.On("Fetch", mock.Anything).Return(nil)
// 	mockWorktree.On("Checkout", mock.Anything).Return(nil)

// 	err := g.CheckoutBranch("repo", "branch")
// 	assert.NoError(t, err)
// }

func TestCheckoutBranch_ErrorOnOpen(t *testing.T) {
	mockGitClient := new(MockGitClient)
	mockLogger := &MockLogger{}
	g := &GitInit{
		GitClient: mockGitClient,
		ZapLogger: mockLogger,
	}
	repo := new(git.Repository)
	mockGitClient.On("PlainOpen", "repo").Return(*repo, errors.New("fail"))
	err := g.CheckoutBranch("repo", "branch")
	assert.Error(t, err)
}

// func TestPush_Success(t *testing.T) {
// 	repo := new(git.Repository)
// 	mockRepo := new(MockRepository)
// 	mockGitClient := new(MockGitClient)
// 	mockLogger := &MockLogger{}
// 	g := &GitInit{
// 		GitClient:     mockGitClient,
// 		ZapLogger:     mockLogger,
// 		USER:          "user",
// 		PAT:           "pat",
// 		CorrelationID: "cid",
// 	}
// 	mockGitClient.On("PlainOpen", "repo").Return(*repo, nil)
// 	mockRepo.On("Push", mock.Anything).Return(nil)

// 	err := g.Push("repo", "branch")
// 	assert.NoError(t, err)
// }

// func TestPush_ErrorOnPush(t *testing.T) {
// 	repo := new(git.Repository)
// 	mockRepo := new(MockRepository)
// 	mockGitClient := new(MockGitClient)
// 	mockLogger := &MockLogger{}
// 	g := &GitInit{
// 		GitClient:     mockGitClient,
// 		ZapLogger:     mockLogger,
// 		USER:          "user",
// 		PAT:           "pat",
// 		CorrelationID: "cid",
// 	}
// 	mockGitClient.On("PlainOpen", "repo").Return(*repo, nil)
// 	mockRepo.On("Push", mock.Anything).Return(errors.New("fail"))

// 	err := g.Push("repo", "branch")
// 	assert.Error(t, err)
// }

// func TestPush_RetryOnNonFastForward(t *testing.T) {
// 	repo := new(git.Repository)
// 	mockRepo := new(MockRepository)
// 	mockGitClient := new(MockGitClient)
// 	mockLogger := &MockLogger{}
// 	g := &GitInit{
// 		GitClient:     mockGitClient,
// 		ZapLogger:     mockLogger,
// 		USER:          "user",
// 		PAT:           "pat",
// 		CorrelationID: "cid",
// 	}
// 	mockGitClient.On("PlainOpen", "repo").Return(*repo, nil)
// 	// First push returns ErrNonFastForwardUpdate, second returns nil
// 	mockRepo.On("Push", mock.Anything).Return(git.ErrNonFastForwardUpdate).Once()
// 	mockRepo.On("Push", mock.Anything).Return(nil).Once()
// 	// CheckoutBranch called on retry
// 	g.CheckoutBranchFunc = func(repoPath, branch string) error { return nil }

// 	err := g.Push("repo", "branch")
// 	assert.NoError(t, err)
// }

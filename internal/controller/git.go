package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CommitAndPushChanges stages, commits, and pushes changes in a Git repository.
func CommitAndPushChanges(repoPath, commitMessage, username, token string, ctx context.Context) error {

	logging := log.FromContext(ctx)

	// Open the existing repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		logging.Error(err, "failed to open git repository")
		return err
	}

	// Get the working tree
	worktree, err := repo.Worktree()
	if err != nil {
		logging.Error(err, "failed to get worktree")
		return err
	}

	// Check for changes in the working tree
	status, err := worktree.Status()
	if err != nil {
		logging.Error(err, "failed to get worktree status")
		return err
	}

	if status.IsClean() {
		logging.Info("No changes to commit.")
		return nil
	}

	// Stage all changes
	err = worktree.AddWithOptions(&git.AddOptions{All: true})
	if err != nil {
		logging.Error(err, "failed to stage changes")
		return err
	}

	// Commit the changes
	_, err = worktree.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  username,
			Email: fmt.Sprintf("%s@ekswatch.devops", username), // Replace with a valid email if needed
			When:  time.Now(),
		},
	})
	if err != nil {
		logging.Error(err, "failed to commit changes")
		return err
	}

	logging.Info("Changes committed successfully.")

	// Push the changes to the origin
	err = repo.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: username, // This can be anything except an empty string
			Password: token,    // Personal Access Token (PAT)
		},
	})
	if err != nil {
		logging.Error(err, "failed to push changes")
		return err
	}

	logging.Info("Changes pushed to repo successfully.")
	return nil
}

package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestDeleteFolder_Success tests that DeleteFolder deletes a folder and its contents.
func TestDeleteFolder_Success(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create a file inside the temp directory
	filePath := filepath.Join(tempDir, "testfile.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Call DeleteFolder
	err := DeleteFolder(tempDir, context.TODO())
	if err != nil {
		t.Errorf("DeleteFolder returned error: %v", err)
	}

	// Check that the directory no longer exists
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("expected directory to be deleted, but it still exists")
	}
}

// TestDeleteFolder_NonExistent tests that DeleteFolder returns no error for non-existent folder.
func TestDeleteFolder_NonExistent(t *testing.T) {
	nonExistentDir := filepath.Join(os.TempDir(), "does-not-exist-12345")

	// Ensure the directory does not exist
	_ = os.RemoveAll(nonExistentDir)

	// Call DeleteFolder
	err := DeleteFolder(nonExistentDir, context.TODO())
	if err != nil {
		t.Errorf("DeleteFolder returned error for non-existent folder: %v", err)
	}
}

// TestDeleteFolder_PermissionDenied tests DeleteFolder when lacking permissions (best effort).
// func TestDeleteFolder_PermissionDenied(t *testing.T) {
// 	// This test is best-effort and may not work on all systems.
// 	tempDir := t.TempDir()
// 	protectedDir := filepath.Join(tempDir, "protected")
// 	if err := os.Mkdir(protectedDir, 0000); err != nil {
// 		t.Skipf("could not create protected dir: %v", err)
// 	}
// 	defer os.Chmod(protectedDir, 0700) // Restore permissions so TempDir cleanup works

// 	err := DeleteFolder(protectedDir, context.TODO())
// 	if err == nil {
// 		t.Errorf("expected error when deleting protected folder, got nil")
// 	}
// }

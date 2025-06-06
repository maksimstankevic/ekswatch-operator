package controller

import (
	"context"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DeleteFolder deletes the specified folder and all its contents.
func DeleteFolder(folderPath string, ctx context.Context) error {

	logging := log.FromContext(ctx)

	err := os.RemoveAll(folderPath)
	if err != nil {
		logging.Error(err, "failed to delete folder")
		return err
	}
	logging.Info("Successfully deleted existing repo folder", "folder", folderPath)
	return nil
}

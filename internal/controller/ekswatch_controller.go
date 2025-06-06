/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ekstoolsv1alpha1 "github.com/maksimstankevic/ekswatch-operator/api/v1alpha1"

	"github.com/maksimstankevic/aws-go/asm"
	mysts "github.com/maksimstankevic/aws-go/sts"

	myeks "github.com/maksimstankevic/aws-go/eks"
)

type Cluster struct {
	Name         string            `yaml:"name"`
	SecretPrefix string            `yaml:"secretPrefix,omitempty"`
	Labels       map[string]string `yaml:"labels"`
}

type ClustersFile struct {
	Clusters []Cluster `yaml:"clusters"`
}

// EkswatchReconciler reconciles a Ekswatch object
type EkswatchReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=ekstools.devops.automation,resources=ekswatches,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ekstools.devops.automation,resources=ekswatches/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ekstools.devops.automation,resources=ekswatches/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Ekswatch object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *EkswatchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logging := log.FromContext(ctx)

	// Fetch the Ekswatch instance
	var ekswatch ekstoolsv1alpha1.Ekswatch
	if err := r.Get(ctx, req.NamespacedName, &ekswatch); err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			logging.Info("ekswatch resource not found. Ignoring since object might have beed deleted")
			return ctrl.Result{}, nil
		}
		logging.Error(err, "unable to fetch Ekswatch")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logging.Info("Fetched Ekswatch", "accounts", ekswatch.Spec.AccountsToWatch)

	logging.Info("Reconciling Ekswatch: " + ekswatch.Name)

	repoDir := "/tmp/ekswatch_repo"

	if len(ekswatch.Spec.ClustersToSyncRegexList) > 0 {
		logging.Info("Clusters to sync regex list", "regexes", ekswatch.Spec.ClustersToSyncRegexList)

		// Check if the repo directory exists
		if _, err := os.Stat(repoDir); err == nil {
			logging.Info("Repo directory exists, deleting it before new clone", "repoDir", repoDir)
			// If the directory exists, delete it
			err := DeleteFolder(repoDir, ctx)
			if err != nil {
				logging.Error(err, "Error deleting repo directory")
				return ctrl.Result{}, err
			}
			logging.Info("Deleted repo directory", "repoDir", repoDir)
		}

		logging.Info("Cloning repository")

		transport.UnsupportedCapabilities = []capability.Capability{
			capability.ThinPack,
		}

		r, err := git.PlainClone(repoDir, false, &git.CloneOptions{
			// The intended use of a GitHub personal access token is in replace of your password
			// because access tokens can easily be revoked.
			// https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/
			Auth: &http.BasicAuth{
				Username: "dummy", // yes, this can be anything except an empty string
				Password: os.Getenv("PAT"),
			},
			URL:   ekswatch.Spec.GitRepository,
			Depth: 1,
		})
		if err != nil {
			logging.Error(err, "Error cloning git repository")
			return ctrl.Result{}, err
		}
		logging.Info("Cloned git repository", "repo", ekswatch.Spec.GitRepository)

		ref, err := r.Head()
		if err != nil {
			logging.Error(err, "Error getting git head")
			return ctrl.Result{}, err
		}

		commit, err := r.CommitObject(ref.Hash())
		if err != nil {
			logging.Error(err, "Error getting git commit object")
			return ctrl.Result{}, err
		}

		logging.Info("Last git commit", "commit", commit)
	}

	var allClusters = make([][]string, len(ekswatch.Spec.AccountsToWatch))
	var allAuthErrors = make([]error, len(ekswatch.Spec.AccountsToWatch))
	var allListingErrors = make([]error, len(ekswatch.Spec.AccountsToWatch))
	wg := sync.WaitGroup{}

	for i, account := range ekswatch.Spec.AccountsToWatch {
		wg.Add(1)
		go func(i int, account ekstoolsv1alpha1.Account) {
			defer wg.Done()
			var sess *session.Session
			sess, allAuthErrors[i] = mysts.GetCredsViaSts(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), account.AccountID, account.RoleName, ctx)
			if sess == nil {
				return
			}
			allListingErrors[i] = myeks.ListEKSClusters(sess, &allClusters[i], ctx)
		}(i, account)
	}

	// Get creds for listing k8s secrets

	sess, err := mysts.GetCredsViaSts(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ekswatch.Spec.K8sSecretsLocation.AccountId, ekswatch.Spec.K8sSecretsLocation.RoleName, ctx)
	if err != nil {
		logging.Error(err, "Error getting STS creds for listing k8s secrets")
		return ctrl.Result{}, err
	}

	var k8sSecrets []string

	wg.Add(1)
	go func(secrets []string, sess *session.Session) error {
		defer wg.Done()
		// Get the list of k8s secrets
		var err error
		k8sSecrets, err = asm.ListSecrets(sess, ekswatch.Spec.K8sSecretsLocation.Region, ctx)
		if err != nil {
			logging.Error(err, "Error listing k8s secrets")
			return err
		}
		return nil
	}(k8sSecrets, sess)

	wg.Wait()

	// All EKS listing goroutines are done - timestamp
	logging.Info("All EKS listing goroutines are done", "time", time.Now())

	// Check for AUTH errors
	for i, err := range allAuthErrors {
		if err != nil {
			logging.Error(err, "Error getting STS creds for account: "+ekswatch.Spec.AccountsToWatch[i].AccountID)
		}
	}

	// Check for LISTING errors
	for i, err := range allListingErrors {
		if err != nil {
			logging.Error(err, "Error listing EKS clusters in account: "+ekswatch.Spec.AccountsToWatch[i].AccountID)
		}
	}

	// List all clusters in all accounts
	logging.Info("All clusters", "clusters", allClusters)

	// List k8s secrets DEBUG ONLY
	// logging.Info("K8s secrets", "secrets", k8sSecrets)

	// Update the status of the Ekswatch instance
	ekswatch.Status.Clusters = make([]ekstoolsv1alpha1.Cluster, 0)
	for i, account := range ekswatch.Spec.AccountsToWatch {
		for _, cluster := range allClusters[i] {
			// Check if the cluster has a secret
			hasSecret := false
			var secrets []string
			secrets = append(secrets, "NoSecretsFound")
			for _, secret := range k8sSecrets {
				if secret == cluster || secret == "k8s-"+cluster {
					secrets = appendSecret(secret, secrets)
					hasSecret = true
				}
			}

			var autoSynced bool = false
			var lastAutoSyncSucceeded string = "n/a"

			// Check if the cluster matches any of the regexes
			matches, err := MatchesAnyRegex(ekswatch.Spec.ClustersToSyncRegexList, cluster, ctx)
			if err != nil {
				logging.Error(err, "Error matching regex for cluster: "+cluster)
				return ctrl.Result{}, err
			}
			if !matches {
				logging.Info("Cluster does not match any regex, not autosyncing", "cluster", cluster)
			} else {
				logging.Info("Cluster matches regex, autosyncing", "cluster", cluster)
				autoSynced = true
				// Add the cluster to the YAML file
				err := AddClusterIfNotExists(repoDir+"/values/clusters.yaml", cluster, ctx)
				if err != nil {
					logging.Error(err, "Error adding cluster to YAML file")
					lastAutoSyncSucceeded = "no"
					return ctrl.Result{}, err
				}
				lastAutoSyncSucceeded = "yes"
			}

			ekswatch.Status.Clusters = append(ekswatch.Status.Clusters, ekstoolsv1alpha1.Cluster{
				Name:   cluster,
				Status: ekstoolsv1alpha1.ClusterStatusActive,
				Account: ekstoolsv1alpha1.Account{
					AccountID: account.AccountID,
					RoleName:  account.RoleName,
				},
				HasSecrets:            hasSecret,
				SecretNames:           secrets,
				AutoSyncEnabled:       autoSynced,
				LastAutoSyncSucceeded: lastAutoSyncSucceeded,
			})
		}
	}
	// Set the status of Ekswatch instance
	if err := r.Status().Update(ctx, &ekswatch); err != nil {
		logging.Error(err, "unable to update Ekswatch status")
		return ctrl.Result{}, err
	}
	logging.Info("Updated Ekswatch status", "status", ekswatch.Status)

	hostname, err := os.Hostname()
	if err != nil {
		logging.Error(err, "Error getting hostname, going with default")
		hostname = "ekswatch-operator"
	}

	// Commit and push changes to the git repository
	if err := CommitAndPushChanges(repoDir, "AutoSync by Ekswatch: "+time.Now().Format("2006-01-02 15:04:05 UTC"), hostname, os.Getenv("PAT"), ctx); err != nil {
		logging.Error(err, "Error committing and pushing changes to git repository")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EkswatchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ekstoolsv1alpha1.Ekswatch{}).
		Complete(r)
}

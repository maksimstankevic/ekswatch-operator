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
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/sts"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ekstoolsv1alpha1 "github.com/maksimstankevic/ekswatch-operator/api/v1alpha1"
)

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

	// TODO(user): your logic here

	logging.Info("Reconciling Ekswatch")

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

	var allClusters = make([][]string, len(ekswatch.Spec.AccountsToWatch))
	var allAuthErrors = make([]error, len(ekswatch.Spec.AccountsToWatch))
	var allListingErrors = make([]error, len(ekswatch.Spec.AccountsToWatch))
	wg := sync.WaitGroup{}

	for i, account := range ekswatch.Spec.AccountsToWatch {
		wg.Add(1)
		go func(i int, account ekstoolsv1alpha1.Account) {
			defer wg.Done()
			var sess *session.Session
			sess, allAuthErrors[i] = getCredsViaSts(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), account.AccountID, account.RoleName)
			if sess == nil {
				return
			}
			allListingErrors[i] = listEKSClusters(sess, &allClusters[i])
		}(i, account)
	}

	// Get creds for listing k8s secrets

	sess, err := getCredsViaSts(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ekswatch.Spec.K8sSecretsLocation.AccountId, ekswatch.Spec.K8sSecretsLocation.RoleName)
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
		k8sSecrets, err = listSecrets(sess, ekswatch.Spec.K8sSecretsLocation.Region)
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

	// List k8s secrets
	logging.Info("K8s secrets", "secrets", k8sSecrets)

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
			ekswatch.Status.Clusters = append(ekswatch.Status.Clusters, ekstoolsv1alpha1.Cluster{
				Name:   cluster,
				Status: ekstoolsv1alpha1.ClusterStatusActive,
				Account: ekstoolsv1alpha1.Account{
					AccountID: account.AccountID,
					RoleName:  account.RoleName,
				},
				HasSecrets:  hasSecret,
				SecretNames: secrets,
			})
		}
	}
	// Set the status of the Ekswatch instance
	if err := r.Status().Update(ctx, &ekswatch); err != nil {
		logging.Error(err, "unable to update Ekswatch status")
		return ctrl.Result{}, err
	}
	logging.Info("Updated Ekswatch status", "status", ekswatch.Status)

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EkswatchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ekstoolsv1alpha1.Ekswatch{}).
		Complete(r)
}

func listEKSClusters(sess *session.Session, clusters *[]string) error {

	// Get all available regions
	ec2Svc := ec2.New(sess)
	regionsOutput, err := ec2Svc.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		return fmt.Errorf("failed to describe regions: %w", err)
	}

	// Iterate over all regions and list EKS clusters
	for _, region := range regionsOutput.Regions {
		regionName := aws.StringValue(region.RegionName)
		eksSvc := eks.New(sess, &aws.Config{Region: aws.String(regionName)})

		listClustersInput := &eks.ListClustersInput{}
		for {
			listClustersOutput, err := eksSvc.ListClusters(listClustersInput)
			if err != nil {
				return fmt.Errorf("failed to list clusters in region %s: %w", regionName, err)
			}

			*clusters = append(*clusters, aws.StringValueSlice(listClustersOutput.Clusters)...)

			if listClustersOutput.NextToken == nil {
				break
			}
			listClustersInput.NextToken = listClustersOutput.NextToken
		}
	}

	return nil
}

func getCredsViaSts(accessKeyID string, secretAccessKey string, accountId string, roleToAssume string) (*session.Session, error) {

	// build complete role ARN
	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, roleToAssume)

	// Create a new AWS session
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
		Region:      aws.String("eu-west-1"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Assume the specified role
	stsSvc := sts.New(sess)
	assumeRoleOutput, err := stsSvc.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         aws.String(roleArn),
		RoleSessionName: aws.String("ekswatch-session"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %w", err)
	}

	// Create a new session with the assumed role credentials
	sess, err = session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(
			aws.StringValue(assumeRoleOutput.Credentials.AccessKeyId),
			aws.StringValue(assumeRoleOutput.Credentials.SecretAccessKey),
			aws.StringValue(assumeRoleOutput.Credentials.SessionToken),
		),
		Region: aws.String("eu-west-1"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session with assumed role: %w", err)
	}
	return sess, nil
}

func listSecrets(sess *session.Session, region string) ([]string, error) {

	// Create a new Secrets Manager client
	svc := secretsmanager.New(sess, &aws.Config{Region: aws.String(region)})

	// List all secrets
	input := &secretsmanager.ListSecretsInput{}
	var secrets []string
	err := svc.ListSecretsPages(input, func(page *secretsmanager.ListSecretsOutput, lastPage bool) bool {
		for _, secret := range page.SecretList {
			secrets = append(secrets, aws.StringValue(secret.Name))
		}
		return !lastPage
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	return secrets, nil

}

func appendSecret(secret string, secrets []string) []string {
	if secrets[0] == "NoSecretsFound" {
		secrets[0] = secret
	} else {
		secrets = append(secrets, secret)
	}
	return secrets
}

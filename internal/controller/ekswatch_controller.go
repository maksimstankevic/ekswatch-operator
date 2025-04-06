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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
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
		logging.Error(err, "unable to fetch Ekswatch")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logging.Info("Fetched Ekswatch", "accounts", ekswatch.Spec.AccountsToWatch)

	for _, account := range ekswatch.Spec.AccountsToWatch {
		logging.Info("Listing EKS clusters for account", "accountID", account.AccountID)

		// List EKS clusters
		clusterNames, err := listEKSClusters(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"))
		if err != nil {
			logging.Error(err, "failed to list EKS clusters")
			return ctrl.Result{}, err
		}

		logging.Info("EKS clusters found", "clusters", clusterNames)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EkswatchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ekstoolsv1alpha1.Ekswatch{}).
		Complete(r)
}

func listEKSClusters(accessKeyID string, secretAccessKey string) ([]string, error) {
	// Create a new AWS session
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
		Region:      aws.String("eu-west-1"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Get all available regions
	ec2Svc := ec2.New(sess)
	regionsOutput, err := ec2Svc.DescribeRegions(&ec2.DescribeRegionsInput{
		RegionNames: []*string{aws.String("eu-west-1")},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe regions: %w", err)
	}

	var clusterNames []string

	// Iterate over all regions and list EKS clusters
	for _, region := range regionsOutput.Regions {
		regionName := aws.StringValue(region.RegionName)
		eksSvc := eks.New(sess, &aws.Config{Region: aws.String(regionName)})

		listClustersInput := &eks.ListClustersInput{}
		for {
			listClustersOutput, err := eksSvc.ListClusters(listClustersInput)
			if err != nil {
				return nil, fmt.Errorf("failed to list clusters in region %s: %w", regionName, err)
			}

			clusterNames = append(clusterNames, aws.StringValueSlice(listClustersOutput.Clusters)...)

			if listClustersOutput.NextToken == nil {
				break
			}
			listClustersInput.NextToken = listClustersOutput.NextToken
		}
	}

	return clusterNames, nil
}

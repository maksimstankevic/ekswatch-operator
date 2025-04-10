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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Account carries AWS Account information and regex for cluster names
// which should be synced to git repository
type Account struct {
	// AccountID is the AWS Account ID
	AccountID string `json:"accountID"`
	// ClusterNameRegex is the regex for cluster names that should be synced
	// to git repository
	ClusterNameRegex []string `json:"clusterNameRegex,omitempty"`
	// RoleName is the name of the role which should be used to assume
	// the role in the target account, if not specified, the default
	// role named ekswatch will be used
	RoleName string `json:"roleName,omitempty"`
}

// Cluster represents an EKS cluster with its name and region
type Cluster struct {
	Name    string        `json:"name"`
	Status  ClusterStatus `json:"status"`
	Account Account       `json:"account"`
}

// EkswatchSpec defines the desired state of Ekswatch
type EkswatchSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// AccountsToWatch is a list of AWS Accounts which should be watched
	// for EKS clusters. The controller will sync the cluster names
	// matching the regex to the git repository.
	AccountsToWatch []Account `json:"accountsToWatch"`
}

// EkswatchStatus defines the observed state of Ekswatch
type EkswatchStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +operator-sdk:csv:customresourcedefinitions.type=status
	// Conditions is an array of current observed cluster conditions
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	// ObservedGeneration is the most recent generation observed for this
	// Ekswatch. It corresponds to the Ekswatch's generation that was
	// most recently processed by the controller.
	// The observed generation may not equal the current generation.
	// For example, when a new Ekswatch is created, the controller may
	// not have finished creating the new Ekswatch's resources by the
	// time the controller's status is updated. In this case, the
	// controller's status will reflect the previous generation.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// LastSyncedAt is the last time the controller synced the cluster names
	LastSyncedAt metav1.Time `json:"lastSyncedAt,omitempty"`
	// LastError is the last error encountered by the controller
	LastError string `json:"lastError,omitempty"`
	// LastErrorTime is the last time the controller encountered an error
	LastErrorTime metav1.Time `json:"lastErrorTime,omitempty"`
	// LastSyncedClusters is the last list of clusters synced to the git repository
	LastSyncedClusters []string `json:"lastSyncedClusters,omitempty"`
	// LastSyncedClustersCount is the last count of clusters synced to the git repository
	LastSyncedClustersCount int `json:"lastSyncedClustersCount,omitempty"`
	// Clusters is the list of clusters in the watched accounts
	Clusters []Cluster `json:"clusters,omitempty"`
	// ClusterCount is the number of clusters in the watched accounts
	ClusterCount int `json:"clusterCount,omitempty"`
	// ClusterCountByAccount is the number of clusters in the watched accounts
	ClusterCountByAccount map[string]int `json:"clusterCountByAccount,omitempty"`
	// ClusterCountByRegion is the number of clusters in the watched accounts
	ClusterCountByRegion map[string]int `json:"clusterCountByRegion,omitempty"`
	// ClusterCountByStatus is the number of clusters in the watched accounts
	ClusterCountByStatus map[string]int `json:"clusterCountByStatus,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Ekswatch is the Schema for the ekswatches API
type Ekswatch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EkswatchSpec   `json:"spec,omitempty"`
	Status EkswatchStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EkswatchList contains a list of Ekswatch
type EkswatchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Ekswatch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Ekswatch{}, &EkswatchList{})
}

/*
Copyright 2021.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VsphereInstanceSpec defines the desired state of VsphereInstance
type VsphereInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of VsphereInstance. Edit vsphere_types.go to remove/update
	CloudID            int               `json:"cloudId,omitempty"`
	GroupID            int               `json:"groupId,omitempty"`
	InstanceTypeCode   string            `json:"instanceTypeCode,omitempty"`
	InstanceTypeLayout int               `json:"instanceLayoutId,omitempty"`
	PlanID             int               `json:"planId,omitempty"`
	Environment        string            `json:"environment,omitempty"`
	ResourcePoolID     int               `json:"resourcePoolId,omitempty"`
	NetworkID          int               `json:"networkId,omitempty"`
	CustomOptions      map[string]string `json:"customOptions,omitempty"`
	//Volumes            []map[interface{}] `json:"volumes,omitempty"`
}

// VsphereInstanceStatus defines the observed state of VsphereInstance
type VsphereInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State      string `json:"state"`
	MorpheusID int    `json:"morpheusId,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="MorpheusID",type="string",JSONPath=".status.morpheusId"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state"
//+kubebuilder:printcolumn:name="Environment",type="string",JSONPath=".spec.environment"
//+kubebuilder:printcolumn:name="InstanceType",type="string",JSONPath=".spec.instanceTypeCode"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// VsphereInstance is the Schema for the vspheres API
type VsphereInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VsphereInstanceSpec   `json:"spec,omitempty"`
	Status VsphereInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VsphereInstanceList contains a list of VsphereInstance
type VsphereInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VsphereInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VsphereInstance{}, &VsphereInstanceList{})
}

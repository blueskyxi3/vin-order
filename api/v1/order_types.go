/*
Copyright 2022 vicentzou.

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

// OrderSpec defines the desired state of Order
type OrderSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Order. Edit order_types.go to remove/update
	Type      string `json:"type,omitempty"`
	OrderNo   string `json:"orderNo,omitempty"`
	CreatedBy string `json:"createdBy,omitempty"`
}

// OrderStatus defines the observed state of Order
type OrderStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Phase defines the current operation that the order process is taking.
	Phase OrderPhase `json:"phase,omitempty"`
	// StartTime is the times that the order entered the `In-progress' phase.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`
	// CompletionTime is the time that the order entered the `Completed' phase.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="CreatedBy",type="string",JSONPath=".spec.createdBy",description="the user who create order"
//+kubebuilder:printcolumn:name="OrderNo",type="string",priority=1,JSONPath=".spec.orderNo",description="the order no"
//+kubebuilder:printcolumn:name="Type",type="string",priority=1,JSONPath=".spec.type",description="the order workflow type"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
//+kubebuilder:subresource:status

// Order is the Schema for the orders API
type Order struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrderSpec   `json:"spec,omitempty"`
	Status OrderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OrderList contains a list of Order
type OrderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Order `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Order{}, &OrderList{})
}

type OrderPhase string

var (
	OrderPhaseInProgress OrderPhase = "In-progress"
	OrderPhaseCompleted  OrderPhase = "Completed"
	OrderPhaseFailed     OrderPhase = "Failed"
)

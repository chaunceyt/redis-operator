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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RedisSpec defines the desired state of Redis
type RedisSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	GlobalConfig GlobalConfig `json:"global"`

	RedisBloom      RedisBloom      `json:"bloom"`
	RedisGraph      RedisGraph      `json:"graph"`
	RedisJSON       RedisJSON       `json:"json"`
	RedisSearch     RedisSearch     `json:"search"`
	RedisTimeSeries RedisTimeSeries `json:"timeSeries"`
}

// GlobalConfig will be the JSON struct for Basic Redis Config
type GlobalConfig struct {
	Password  string                      `json:"password,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type RedisBloom struct {
	Enabled bool `json:"enabled"`
}
type RedisTimeSeries struct {
	Enabled bool `json:"enabled"`
}

type RedisGraph struct {
	Enabled bool `json:"enabled"`
}

type RedisJSON struct {
	Enabled bool `json:"enabled"`
}

type RedisSearch struct {
	Enabled bool `json:"enabled"`
}

// RedisExporter interface will have the information for redis exporter related stuff
type RedisExporter struct {
	Enabled         bool                        `json:"enabled,omitempty"`
	Image           string                      `json:"image"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`
	ImagePullPolicy corev1.PullPolicy           `json:"imagePullPolicy,omitempty"`
}

type Resources struct {
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// RedisStatus defines the observed state of Redis
type RedisStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Nodes []string `json:"nodes"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Redis is the Schema for the redis API
type Redis struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisSpec   `json:"spec,omitempty"`
	Status RedisStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RedisList contains a list of Redis
type RedisList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Redis `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Redis{}, &RedisList{})
}

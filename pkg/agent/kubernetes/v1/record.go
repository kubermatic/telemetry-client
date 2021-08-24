/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Telemetry takes the Record schema from Kubernetes spartakus project:
https://github.com/kubernetes-retired/spartakus/blob/master/pkg/volunteer/kubernetes.go
and some customized fields are added to fit Telemetry own use cases.
*/

package v1

import (
	"fmt"
	"time"

	"github.com/kubermatic/telemetry-client/pkg/agent"
)

type Record struct {
	agent.KindVersion
	// Time is the time when the record is generated.
	Time time.Time `json:"time"`
	// Kubernetes version of this cluster.
	KubernetesVersion string `json:"kubernetes_version"`
	// Nodes is a list of node-specific information from the reporting cluster.
	Nodes []Node `json:"nodes,omitempty"`
}

func (r *Record) String() string {
	return fmt.Sprintf("Record kind: %s version: %s", r.Kind, r.Version)
}

type Node struct {
	// ID is a unique string that identifies a node in tis cluster.  It can be
	// any value but we strongly recommend a random GUID or a hash derived from
	// identifying information.  This should be a stable value for the lifetime
	// of the node, or else it will be assumed to be a different node.  This
	// must not include personally identifiable information.
	ID string `json:"id"`
	// OperatingSystem is the value reported by kubernetes in the node status.
	OperatingSystem *string `json:"operating_system,omitempty"`
	// OSImage is the value reported by kubernetes in the node status.
	OSImage *string `json:"os_image,omitempty"`
	// KernelVersion is the value reported by kubernetes in the node status.
	KernelVersion *string `json:"kernel_version,omitempty"`
	// Architecture is the value reported by kubernetes in the node status.
	Architecture *string `json:"architecture,omitempty"`
	// ContainerRuntimeVersion is the value reported by kubernetes in the node
	// status.
	ContainerRuntimeVersion *string `json:"container_runtime_version,omitempty"`
	// KubeletVersion is the value reported by kubernetes in the node status.
	KubeletVersion *string `json:"kubelet_version,omitempty"`
	// CloudProvider is the <ProviderName> portion of the ProviderID reported
	// by kubernetes in the node spec.
	CloudProvider *string `json:"cloud_provider,omitempty"`
	// Capacity is a list of resources and their associated values as reported
	// by kubernetes in the node status.
	Capacity []Resource `json:"capacity,omitempty"`
}

type Resource struct {
	// Resource is the name of the resource.
	Resource string `json:"resource"` // required
	// Value is the string form of the of the resource's value.
	Value string `json:"value"` // required
}

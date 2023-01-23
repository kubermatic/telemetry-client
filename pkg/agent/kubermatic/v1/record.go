/*
Copyright 2021 The Telemetry Authors.

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
	// Seeds is a list of seed-specific information.
	Seeds []Seed `json:"seeds,omitempty"`
	// Clusters is a list of cluster-specific information.
	Clusters []Cluster `json:"clusters,omitempty"`
	// Users is a list of users
	Users []User `json:"users,omitempty"`
	// Projects is a list of projects
	Projects []Project `json:"projects,omitempty"`
	// SSHKeys is a list of SSHKeys
	SSHKeys []SSHKey `json:"ssh_keys,omitempty"`
}

func (r *Record) String() string {
	return fmt.Sprintf("Record kind: %s version: %s", r.Kind, r.Version)
}

type Seed struct {
	UUID string `json:"uuid,omitempty"`
	// Country of the seed as ISO-3166 two-letter code, e.g. DE or UK.
	Country string `json:"country,omitempty"`
	// Detailed location of the cluster, like "Hamburg" or "Datacenter 7".
	Location string `json:"location,omitempty"`
	// ExposeStrategy explicitly sets the expose strategy for this seed cluster,
	// if not set, the default provided by the master is used.
	ExposeStrategy string `json:"expose_strategy,omitempty"`
	// Datacenters contains a list of the possible datacenters (DCs) in this seed.
	// Each DC must have a globally unique identifier (i.e. names must be unique
	// across all seeds).
	Datacenters []Datacenter `json:"datacenters,omitempty"`
}

// Datacenter specifies the data for a datacenter.
type Datacenter struct {
	UUID string `json:"uuid,omitempty"`
	// Country of the seed as ISO-3166 two-letter code, e.g. DE or UK.
	Country string `json:"country,omitempty"`
	// Detailed location of the cluster, like "Hamburg" or "Datacenter 7".
	Location string `json:"location,omitempty"`
	// Provider contains the cloud provider name used to manage resources
	// in this datacenter.
	Provider string `json:"provider,omitempty"`
	// Region contains cloud provider region for this datacenter.
	Region string `json:"region,omitempty"`
}

type Cluster struct {
	UUID string `json:"uuid,omitempty"`

	// SeedUUID helps to uniquely relate this cluster with the owned seed
	SeedUUID string `json:"seed_uuid,omitempty"`

	// ProjectUUID helps to uniquely relate this cluster with the owned project
	ProjectUUID string `json:"project_uuid,omitempty"`

	// ExposeStrategy is the approach we use to expose this cluster, either via NodePort
	// or via a dedicated LoadBalancer
	ExposeStrategy string `json:"expose_strategy,omitempty"`

	EtcdClusterSize int `json:"etcd_cluster_size,omitempty"`

	// Version defines the wanted version of the control plane
	KubernetesServerVersion string `json:"kubernetes_server_version,omitempty"`

	// KubermaticVersion current kubermatic version.
	KubermaticVersion string `json:"kubermatic_version,omitempty"`

	// Cloud specifies the cloud providers configuration
	Cloud Cloud `json:"cloud,omitempty"`

	// OPAIntegration is a preview feature that enables OPA integration with Kubermatic for the cluster.
	OPAIntegrationEnabled bool `json:"opa_integration_enabled"`

	// MLA contains monitoring, logging and alerting related settings for the user cluster.
	MLA MLASettings `json:"mla,omitempty"`

	// EnableUserSSHKeyAgent control whether the UserSSHKeyAgent will be deployed in the user cluster or not.
	UserSSHKeyAgentEnabled bool `json:"user_ssh_key_agent_enabled"`
}

type Cloud struct {
	ProviderName   string `json:"provider_name,omitempty"`
	DatacenterUUID string `json:"datacenter_uuid,omitempty"`
}

type MLASettings struct {
	// MonitoringEnabled is the flag for enabling monitoring in user cluster.
	MonitoringEnabled bool `json:"monitoring_enabled"`
	// LoggingEnabled is the flag for enabling logging in user cluster.
	LoggingEnabled bool `json:"logging_enabled"`
}

type Project struct {
	UUID string `json:"uuid,omitempty"`
}

type User struct {
	UUID string `json:"uuid,omitempty"`
	// IsAdmin indicates admin role
	IsAdmin bool `json:"is_admin"`
}

type SSHKey struct {
	UUID             string   `json:"uuid,omitempty"`
	OwnerProjectUUID string   `json:"owner_project_uuid,omitempty"`
	ClusterUUIDs     []string `json:"cluster_uuids,omitempty"`
}

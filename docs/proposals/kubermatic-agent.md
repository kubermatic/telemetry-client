# Temetry Kubermatic Agent Proposal

**Author**: Harshita Sharma

**Status**: Draft proposal;

## Motivation and Background
Usage tracker for Kubermatic product usage.

## Goals
- Track open-source Kubermatic products usage
- Generic solution for all our products
- Anonymous (UUID)
- Opt-out (Addon)
- Easy to use for data analysis, machine learning, etc.

### Agent
Kubermatic Agent is a component which collects the data based on the predefined report schema. Agent will collect data as an initContainer and write data to local storage.

### Reporter
Reporter will aggregate data which was collected by the Agent from local storage, and send it to storage backend endpoint.

**Report Schema**

```
type Record struct {
	agent.KindVersion
	// Time is the time when the record is generated.
	Time              time.Time  `json:"time"`
	// Kubernetes version of this cluster.
	KubernetesVersion string     `json:"kubernetesVersion"`
	// Seeds is a list of seed-specific information.
	Seeds             []Seed     `json:"seeds,omitempty"`
	// Clusters is a list of seed-specific information.
	Clusters          []Cluster  `json:"clusters,omitempty"`
	// Users is a list of users
	Users             []User     `json:"users,omitempty"`
	// Projects is a list of projects
	Projects          []Project  `json:"projects,omitempty"`
	// SSHKeys is a list of SSHKeys
	SSHKeys           []SSHKey   `json:"sshKeys,omitempty"`
}
```

**Seeds**:

- UUID
- Country
- Location
- Seed Expose Strategy
- SeedDatacenters: To identify strategic locations of better service points/locations (ex: CDN, on-sight consultancy, prefer creating cluster in same DC for faster network and close to Seed location, cheaper data transfer)
- MLAEnabled

```
type Seed struct {
        UUID           string           `json:"uuid,omitempty"`
	Country        string           `json:"country,omitempty"`
	Location       string           `json:"location,omitempty"`
	ExposeStrategy string           `json:"exposeStrategy,omitempty"`
	Datacenters    []Datacenter     `json:"datacenters,omitempty"`
	MLAEnabled     bool             `json:"userClusterMlaEnabled"`
}

// Datacenter specifies the data for a datacenter.
type Datacenter struct {
        UUID     string `json:"uuid,omitempty"`
	Country  string `json:"country,omitempty"`
	Location string `json:"location,omitempty"`
	Provider string `json:"provider,omitempty"`
	Region   string `json:"region,omitempty"`
}
```

**Clusters**:

Basics:

- UUID
- Expose Strategy
- EtcdClusterSize
- KubernetesServerVersion
- KubermaticVersion
- ContainerRuntime
- Cloud:
    - Provider
    - Datacenter Region

**Features**:

Features being used: Identify features being used by customers to drive and prioritize future development + modifications

- SSH Key Agent (1/0)
- OPA Integration (1/0)
- MLA (1/0)

```
Clusters []Cluster

type Cluster struct{
    UUID                    string     `json:"uuid,omitempty"`

    SeedUUID                string     `json:"seedUUID,omitempty"

    ProjectUUID             string     `json:"projectUUID,omitempty"`

    ExposeStrategy          string     `json:"exposeStrategy,omitempty"`

    EtcdClusterSize         int        `json:"etcdClusterSize,omitempty"`

    // Version defines the wanted version of the control plane
    KubernetesServerVersion string     `json:"kubernetesServerVersion,omitempty"`

    // KubermaticVersion current kubermatic version.
    KubermaticVersion      string      `json:"kubermaticVersion,omitempty"`

    // ContainerRuntime to use, i.e. Docker or containerd.
    ContainerRuntime      string       `json:"containerRuntime,omitempty"`

    // Cloud specifies the cloud providers configuration
    Cloud                 Cloud        `json:"cloud"`

    // OPAIntegration is a preview feature that enables OPA integration with Kubermatic for the cluster.
    OPAIntegrationEnabled bool         `json:"opaIntegrationEnabled"`

    // MLA contains monitoring, logging and alerting related settings for the user cluster.
    MLA                   MLASettings  `json:"mla,omitempty"`

    EnableUserSSHKeyAgent bool         `json:"enableUserSSHKeyAgent"`
}

type Cloud struct{
    Provider       string     `json:"provider,omitempty"`
    DatacenterUUID string     `json:"dcUUID,omitempty"`
}

type MLASettings struct {
    // MonitoringEnabled is the flag for enabling monitoring in user cluster.
    MonitoringEnabled bool    `json:"monitoringEnabled"`
    // LoggingEnabled is the flag for enabling logging in user cluster.
    LoggingEnabled    bool    `json:"loggingEnabled"`
}
```

**Machine Deployment**

**Note**: Will be implemented later with userCluster Client

- OS
- KubeletVersion (*string)
- TotalNumberOfReplica (number of nodes) (int)

```
machineDeployments []machineDeployment

type machineDeployments struct{
    OS string `json:"os,omitempty"`
    KubeletVersion string `json:"kubeletVersion,omitempty"`
    NumberOfReplicas int `json:"numberOfReplicas,omitempty"`
}
```

**SSH Keys**
- Number of SSH Keys per Project:

```
SSHKeys []SSHKey `json:"sshkeys,omitempty"`

type SSHKey struct {
	UUID             string      `json:"uuid,omitempty"`
	OwnerProjectUUID []string    `json:"ownerProjectUUID,omitempty"`
}
```

**Users**:
- Number of users (int)
- Number of Admin users (int)

```
// Users is a list of users
    Users []User `json:"users,omitempty"`

type User struct {
    UUID    string `json:"uuid,omitempty"`
    // IsAdmin indicates admin role
    IsAdmin bool   `json:"isAdmin"`
}
```

**Projects**:
- Number of projects (int)

```
Projects []Project

type Project struct{
    UUID string `json:"uuid,omitempty"`
}
```

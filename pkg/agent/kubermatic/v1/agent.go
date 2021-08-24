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
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kubermatic/telemetry-client/pkg/agent"
	"github.com/kubermatic/telemetry-client/pkg/datastore"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	uuidSpace = "kubermatic"
)

type serverVersionInfo interface {
	ServerVersion() (*version.Info, error)
}

type kubermaticAgent struct {
	client.Client
	serverVersionInfo
	dataStore datastore.DataStore
}

func NewAgent(client client.Client, info serverVersionInfo, dataStore datastore.DataStore) agent.Agent {
	return kubermaticAgent{
		Client:            client,
		serverVersionInfo: info,
		dataStore:         dataStore,
	}
}

// +kubebuilder:rbac:groups="kubermatic.k8s.io",resources=seeds;clusters;users;projects;usersshkeies,verbs=list
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
// +kubebuilder:rbac:groups="operator.kubermatic.io",resources=kubermaticconfigurations,verbs=list

func (a kubermaticAgent) Collect() error {
	ctx := context.Background()
	serverVersion, err := a.ServerVersion()
	if err != nil {
		return err
	}
	record := Record{
		KindVersion: agent.KindVersion{
			Kind:    "kubermatic",
			Version: "v1",
		},
		Time:              time.Now().UTC(),
		KubernetesVersion: serverVersion.String(),
	}

	defaultExposeStrategy, err := a.getDefaultExposeStrategy(ctx)
	if err != nil {
		return err
	}

	// List projects
	projectList := &kubermaticv1.ProjectList{}
	if err := a.List(ctx, projectList); err != nil {
		return fmt.Errorf("failed listing projects: %w", err)
	}

	for _, project := range projectList.Items {
		project, err := projectFromKube(project)
		if err != nil {
			return err
		}
		record.Projects = append(record.Projects, project)
	}

	// List users
	userList := &kubermaticv1.UserList{}
	if err := a.List(ctx, userList); err != nil {
		return fmt.Errorf("failed listing users: %w", err)

	}

	for _, user := range userList.Items {
		user, err := userKeyFromKube(user)
		if err != nil {
			return err
		}
		record.Users = append(record.Users, user)
	}

	// List sshKeys
	sshKeyList := &kubermaticv1.UserSSHKeyList{}
	if err := a.List(ctx, sshKeyList); err != nil {
		return fmt.Errorf("failed listing ssh keys: %w", err)
	}

	for _, sshKey := range sshKeyList.Items {
		sshKey, err := sshKeyFromKube(sshKey)
		if err != nil {
			return err
		}
		record.SSHKeys = append(record.SSHKeys, sshKey)
	}

	// List seeds
	seedList := &kubermaticv1.SeedList{}
	if err := a.List(ctx, seedList); err != nil {
		return fmt.Errorf("failed listing seeds: %w", err)
	}
	for _, seed := range seedList.Items {

		seedKubeconfigGetter, err := provider.SeedKubeconfigGetterFactory(ctx, a.Client)
		if err != nil {
			return err
		}
		seedClientGetter := provider.SeedClientGetterFactory(seedKubeconfigGetter)
		seedClient, err := seedClientGetter(&seed)
		if err != nil {
			return fmt.Errorf("failed getting seed client for seed %s: %w", seed.Name, err)
		}

		//  List clusters per seed
		clusterList := &kubermaticv1.ClusterList{}
		if err := seedClient.List(ctx, clusterList); err != nil {
			return fmt.Errorf("failed listing clusters: %w", err)
		}

		for _, cluster := range clusterList.Items {
			cluster, err := clusterFromKube(cluster, seed.Name)
			if err != nil {
				return err
			}
			record.Clusters = append(record.Clusters, cluster)
		}

		seed, err := seedFromKube(seed, defaultExposeStrategy)
		if err != nil {
			return err
		}
		record.Seeds = append(record.Seeds, seed)

	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return a.dataStore.Store(data)
}

func (a kubermaticAgent) getDefaultExposeStrategy(ctx context.Context) (kubermaticv1.ExposeStrategy, error) {
	kubermaticConfigs := &operatorv1alpha1.KubermaticConfigurationList{}
	if err := a.List(ctx, kubermaticConfigs); err != nil {
		return "", fmt.Errorf("failed listing kubermaitc configurations: %w", err)
	}
	configLen := len(kubermaticConfigs.Items)
	if configLen == 0 || configLen > 1 {
		return "", fmt.Errorf("kubermaitc configuration number not as expected: %v", configLen)
	}

	defaultExposeStrategy := kubermaticConfigs.Items[0].Spec.ExposeStrategy
	if defaultExposeStrategy == "" {
		defaultExposeStrategy = common.DefaultExposeStrategy
	}

	return defaultExposeStrategy, nil
}

func seedFromKube(kSeed kubermaticv1.Seed, defaultExposeStrategy kubermaticv1.ExposeStrategy) (Seed, error) {
	var kDatacenter []Datacenter

	datacenters := kSeed.Spec.Datacenters
	for name, datacenter := range datacenters {
		providerName, err := provider.DatacenterCloudProviderName(&datacenter.Spec)
		if err != nil {
			return Seed{}, err
		}

		kDatacenter = append(kDatacenter, Datacenter{
			UUID:     generateUUID(name),
			Country:  datacenter.Country,
			Location: datacenter.Location,
			Provider: providerName,
			Region:   datacenterCloudRegionName(&datacenter.Spec, providerName),
		})
	}

	var exposeStrategy kubermaticv1.ExposeStrategy = kSeed.Spec.ExposeStrategy
	if exposeStrategy == "" {
		exposeStrategy = defaultExposeStrategy
	}
	seed := Seed{
		UUID:           generateUUID(kSeed.Name),
		Country:        kSeed.Spec.Country,
		Location:       kSeed.Spec.Location,
		ExposeStrategy: string(exposeStrategy),
		Datacenters:    kDatacenter,
	}

	return seed, nil
}

func clusterFromKube(kn kubermaticv1.Cluster, seedName string) (Cluster, error) {
	providerName, err := provider.ClusterCloudProviderName(kn.Spec.Cloud)
	if err != nil {
		return Cluster{}, err
	}

	var opaEnabled bool
	opaIntegration := kn.Spec.OPAIntegration
	if opaIntegration != nil {
		opaEnabled = opaIntegration.Enabled
	}

	var enableUserSSHKeyAgent bool
	enableUserSSHKeyAgentPointer := kn.Spec.EnableUserSSHKeyAgent
	if kn.Spec.EnableUserSSHKeyAgent != nil {
		enableUserSSHKeyAgent = *enableUserSSHKeyAgentPointer
	}

	var mla MLASettings
	mlaSetting := kn.Spec.MLA
	if mlaSetting != nil {
		mla = MLASettings(*mlaSetting)
	}

	cluster := Cluster{
		UUID:                    generateUUID(kn.Name),
		SeedUUID:                generateUUID(seedName),
		ProjectUUID:             generateUUID(kn.Labels[kubermaticv1.ProjectIDLabelKey]),
		ExposeStrategy:          string(kn.Spec.ExposeStrategy),
		EtcdClusterSize:         kn.Spec.ComponentsOverride.Etcd.ClusterSize,
		KubernetesServerVersion: kn.Spec.Version.String(),
		KubermaticVersion:       kn.Status.KubermaticVersion,
		Cloud: Cloud{
			ProviderName:   providerName,
			DatacenterUUID: generateUUID(kn.Spec.Cloud.DatacenterName),
		},
		OPAIntegrationEnabled:  opaEnabled,
		UserSSHKeyAgentEnabled: enableUserSSHKeyAgent,
		MLA:                    mla,
	}
	return cluster, nil
}

func projectFromKube(kn kubermaticv1.Project) (Project, error) {
	project := Project{
		UUID: generateUUID(kn.Name),
	}
	return project, nil
}

func sshKeyFromKube(kn kubermaticv1.UserSSHKey) (SSHKey, error) {
	var ownerProject string
	for _, ownerReference := range kn.OwnerReferences {
		if ownerReference.Kind == kubermaticv1.ProjectKindName {
			ownerProject = generateUUID(ownerReference.Name)
			break
		}
	}

	var clusters []string
	for _, cluster := range kn.Spec.Clusters {
		clusters = append(clusters, generateUUID(cluster))
	}

	SSHKey := SSHKey{
		UUID:             generateUUID(kn.Name),
		OwnerProjectUUID: ownerProject,
		ClusterUUIDs:     clusters,
	}
	return SSHKey, nil
}

func userKeyFromKube(kn kubermaticv1.User) (User, error) {
	user := User{
		UUID:    generateUUID(kn.Name),
		IsAdmin: kn.Spec.IsAdmin,
	}
	return user, nil
}

func generateUUID(x string) string {
	space, _ := uuid.Parse(uuidSpace)
	UUID := uuid.NewMD5(space, []byte(x))

	return UUID.String()
}

func datacenterCloudRegionName(spec *kubermaticv1.DatacenterSpec, providerName string) string {
	if spec == nil {
		return ""
	}

	var region string

	switch providerName {
	case provider.DigitaloceanCloudProvider:
		region = spec.Digitalocean.Region
	case provider.AWSCloudProvider:
		region = spec.AWS.Region
	case provider.AzureCloudProvider:
		region = spec.Azure.Location
	case provider.OpenstackCloudProvider:
		region = spec.Openstack.Region
	case provider.HetznerCloudProvider:
		region = spec.Hetzner.Location
	case provider.VSphereCloudProvider:
		region = spec.VSphere.Datacenter
	case provider.GCPCloudProvider:
		region = spec.GCP.Region
	case provider.AlibabaCloudProvider:
		region = spec.Alibaba.Region
	case provider.AnexiaCloudProvider:
		region = spec.Anexia.LocationID
	}

	return region
}

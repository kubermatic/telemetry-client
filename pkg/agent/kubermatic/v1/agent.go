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

	"github.com/kubermatic/telemetry-client/pkg/agent"
	"github.com/kubermatic/telemetry-client/pkg/datastore"

	"github.com/google/uuid"
	"go.uber.org/zap"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type serverVersionInfo interface {
	ServerVersion() (*version.Info, error)
}

type kubermaticAgent struct {
	client.Client
	serverVersionInfo

	dataStore datastore.DataStore
	log       *zap.SugaredLogger
}

func NewAgent(client client.Client, info serverVersionInfo, dataStore datastore.DataStore, log *zap.SugaredLogger) agent.Agent {
	return kubermaticAgent{
		Client:            client,
		serverVersionInfo: info,
		dataStore:         dataStore,
		log:               log,
	}
}

// +kubebuilder:rbac:groups="kubermatic.k8c.io",resources=seeds;clusters;users;projects;usersshkeys;kubermaticconfigurations,verbs=list
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get

func (a kubermaticAgent) Collect(ctx context.Context) error {
	record := Record{
		KindVersion: agent.KindVersion{
			Kind:    "kubermatic",
			Version: "v1",
		},
		Time: time.Now().UTC(),
	}

	// Get Kubermatic Configuration
	configGetter, err := provider.DynamicKubermaticConfigurationGetterFactory(a.Client, resources.KubermaticNamespace)
	if err != nil {
		return err
	}
	config, err := configGetter(ctx)
	if err != nil {
		return err
	}
	// Get Kubermatic Configuration fields
	record.KubermaticEdition = config.Status.KubermaticEdition
	record.KubermaticVersion = config.Status.KubermaticVersion

	defaultExposeStrategy := config.Spec.ExposeStrategy
	if defaultExposeStrategy == "" {
		defaultExposeStrategy = defaults.DefaultExposeStrategy
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

	a.log.Infow("Collected projects", "projects", len(record.Projects))

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

	a.log.Infow("Collected users", "users", len(record.Users))

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

	a.log.Infow("Collected SSH keys", "keys", len(record.SSHKeys))

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

		a.log.Infow("Collected userclusters", "seed", seed.Name, "clusters", len(record.Clusters))

		seed, err := seedFromKube(seed, defaultExposeStrategy)
		if err != nil {
			return err
		}
		record.Seeds = append(record.Seeds, seed)
	}

	a.log.Infow("Collected seeds", "seeds", len(record.Seeds))

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return a.dataStore.Store(ctx, data)
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

	var cniPlugin CNIPluginSettings
	if kn.Spec.CNIPlugin != nil {
		cniPlugin.Type = kn.Spec.CNIPlugin.Type.String()
		cniPlugin.Version = kn.Spec.CNIPlugin.Version
	}

	var clusterNetworkingConfig ClusterNetworkingConfig
	clusterNetwork := kn.Spec.ClusterNetwork
	clusterNetworkingConfig.IPFamily = string(clusterNetwork.IPFamily)
	if clusterNetwork.KonnectivityEnabled != nil {
		clusterNetworkingConfig.KonnectivityEnabled = *clusterNetwork.KonnectivityEnabled
	}

	var opaEnabled bool
	opaIntegration := kn.Spec.OPAIntegration
	if opaIntegration != nil {
		opaEnabled = opaIntegration.Enabled
	}

	var userSSHKeyAgentEnabled bool
	enableUserSSHKeyAgentPointer := kn.Spec.EnableUserSSHKeyAgent
	if enableUserSSHKeyAgentPointer != nil {
		userSSHKeyAgentEnabled = *enableUserSSHKeyAgentPointer
	}

	var mla MLASettings
	mlaSetting := kn.Spec.MLA
	if mlaSetting != nil {
		mla.MonitoringEnabled = mlaSetting.MonitoringEnabled
		mla.LoggingEnabled = mlaSetting.LoggingEnabled
	}

	etcdSize := 0
	if kn.Spec.ComponentsOverride.Etcd.ClusterSize != nil {
		etcdSize = int(*kn.Spec.ComponentsOverride.Etcd.ClusterSize)
	}
	cluster := Cluster{
		UUID:                    generateUUID(kn.Name),
		SeedUUID:                generateUUID(seedName),
		ProjectUUID:             generateUUID(kn.Labels[kubermaticv1.ProjectIDLabelKey]),
		CNIPlugin:               cniPlugin,
		ClusterNetwork:          clusterNetworkingConfig,
		ExposeStrategy:          string(kn.Spec.ExposeStrategy),
		EtcdClusterSize:         etcdSize,
		KubernetesServerVersion: kn.Spec.Version.String(),
		Cloud: Cloud{
			ProviderName:   providerName,
			DatacenterUUID: generateUUID(kn.Spec.Cloud.DatacenterName),
		},
		OPAIntegrationEnabled:  opaEnabled,
		UserSSHKeyAgentEnabled: userSSHKeyAgentEnabled,
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
	UUID := uuid.NewMD5(uuid.Nil, []byte(x))

	return UUID.String()
}

func datacenterCloudRegionName(spec *kubermaticv1.DatacenterSpec, providerName string) string {
	if spec == nil {
		return ""
	}

	var region string

	switch kubermaticv1.ProviderType(providerName) {
	case kubermaticv1.DigitaloceanCloudProvider:
		region = spec.Digitalocean.Region
	case kubermaticv1.AWSCloudProvider:
		region = spec.AWS.Region
	case kubermaticv1.AzureCloudProvider:
		region = spec.Azure.Location
	case kubermaticv1.OpenstackCloudProvider:
		region = spec.Openstack.Region
	case kubermaticv1.HetznerCloudProvider:
		region = spec.Hetzner.Location
	case kubermaticv1.VSphereCloudProvider:
		region = spec.VSphere.Datacenter
	case kubermaticv1.GCPCloudProvider:
		region = spec.GCP.Region
	case kubermaticv1.AlibabaCloudProvider:
		region = spec.Alibaba.Region
	case kubermaticv1.AnexiaCloudProvider:
		region = spec.Anexia.LocationID
	}

	return region
}

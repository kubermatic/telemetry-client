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
	v1types "github.com/kubermatic/telemetry-client/pkg/agent/kubermatic/v1/types"
	"github.com/kubermatic/telemetry-client/pkg/datastore"
	telemetryversion "github.com/kubermatic/telemetry-client/pkg/version"

	"github.com/google/uuid"
	"go.uber.org/zap"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
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
	serverVersion, err := a.ServerVersion()
	if err != nil {
		return err
	}
	record := v1types.Record{
		KindVersion: agent.KindVersion{
			Kind:    "kubermatic",
			Version: telemetryversion.V1Version,
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
		seedKubeconfigGetter, err := kubernetesprovider.SeedKubeconfigGetterFactory(ctx, a.Client)
		if err != nil {
			return err
		}
		seedClientGetter := kubernetesprovider.SeedClientGetterFactory(seedKubeconfigGetter)
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

func (a kubermaticAgent) getDefaultExposeStrategy(ctx context.Context) (kubermaticv1.ExposeStrategy, error) {
	kubermaticConfigs := &kubermaticv1.KubermaticConfigurationList{}
	if err := a.List(ctx, kubermaticConfigs); err != nil {
		return "", fmt.Errorf("failed listing kubermaitc configurations: %w", err)
	}
	configLen := len(kubermaticConfigs.Items)
	if configLen == 0 || configLen > 1 {
		return "", fmt.Errorf("kubermaitc configuration number not as expected: %v", configLen)
	}

	defaultExposeStrategy := kubermaticConfigs.Items[0].Spec.ExposeStrategy
	if defaultExposeStrategy == "" {
		defaultExposeStrategy = defaulting.DefaultExposeStrategy
	}

	return defaultExposeStrategy, nil
}

func seedFromKube(kSeed kubermaticv1.Seed, defaultExposeStrategy kubermaticv1.ExposeStrategy) (v1types.Seed, error) {
	var kDatacenter []v1types.Datacenter

	datacenters := kSeed.Spec.Datacenters
	for name, datacenter := range datacenters {
		providerName, err := kubermaticv1helper.DatacenterCloudProviderName(&datacenter.Spec)
		if err != nil {
			return v1types.Seed{}, err
		}

		kDatacenter = append(kDatacenter, v1types.Datacenter{
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
	seed := v1types.Seed{
		UUID:           generateUUID(kSeed.Name),
		Country:        kSeed.Spec.Country,
		Location:       kSeed.Spec.Location,
		ExposeStrategy: string(exposeStrategy),
		Datacenters:    kDatacenter,
	}

	return seed, nil
}

func clusterFromKube(kn kubermaticv1.Cluster, seedName string) (v1types.Cluster, error) {
	providerName, err := kubermaticv1helper.ClusterCloudProviderName(kn.Spec.Cloud)
	if err != nil {
		return v1types.Cluster{}, err
	}

	var opaEnabled bool
	opaIntegration := kn.Spec.OPAIntegration
	if opaIntegration != nil {
		opaEnabled = opaIntegration.Enabled
	}

	var enableUserSSHKeyAgent bool
	enableUserSSHKeyAgentPointer := kn.Spec.EnableUserSSHKeyAgent
	if enableUserSSHKeyAgentPointer != nil {
		enableUserSSHKeyAgent = *enableUserSSHKeyAgentPointer
	}

	var mla v1types.MLASettings
	mlaSetting := kn.Spec.MLA
	if mlaSetting != nil {
		mla.MonitoringEnabled = mlaSetting.MonitoringEnabled
		mla.LoggingEnabled = mlaSetting.LoggingEnabled
	}

	etcdSize := 0
	if kn.Spec.ComponentsOverride.Etcd.ClusterSize != nil {
		etcdSize = int(*kn.Spec.ComponentsOverride.Etcd.ClusterSize)
	}
	cluster := v1types.Cluster{
		UUID:                    generateUUID(kn.Name),
		SeedUUID:                generateUUID(seedName),
		ProjectUUID:             generateUUID(kn.Labels[kubermaticv1.ProjectIDLabelKey]),
		ExposeStrategy:          string(kn.Spec.ExposeStrategy),
		EtcdClusterSize:         etcdSize,
		KubernetesServerVersion: kn.Spec.Version.String(),
		Cloud: v1types.Cloud{
			ProviderName:   providerName,
			DatacenterUUID: generateUUID(kn.Spec.Cloud.DatacenterName),
		},
		OPAIntegrationEnabled:  opaEnabled,
		UserSSHKeyAgentEnabled: enableUserSSHKeyAgent,
		MLA:                    mla,
	}
	return cluster, nil
}

func projectFromKube(kn kubermaticv1.Project) (v1types.Project, error) {
	project := v1types.Project{
		UUID: generateUUID(kn.Name),
	}
	return project, nil
}

func sshKeyFromKube(kn kubermaticv1.UserSSHKey) (v1types.SSHKey, error) {
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

	SSHKey := v1types.SSHKey{
		UUID:             generateUUID(kn.Name),
		OwnerProjectUUID: ownerProject,
		ClusterUUIDs:     clusters,
	}
	return SSHKey, nil
}

func userKeyFromKube(kn kubermaticv1.User) (v1types.User, error) {
	user := v1types.User{
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

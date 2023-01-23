/*
Copyright 2020 The Telemetry Authors.

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
	"sort"
	"time"

	"github.com/kubermatic/telemetry-client/pkg/agent"
	"github.com/kubermatic/telemetry-client/pkg/agent/kubernetes"
	"github.com/kubermatic/telemetry-client/pkg/datastore"
	telemetryversion "github.com/kubermatic/telemetry-client/pkg/version"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type serverVersionInfo interface {
	ServerVersion() (*version.Info, error)
}

type kubernetesAgent struct {
	client.Client
	serverVersionInfo
	dataStore datastore.DataStore
	log       *zap.SugaredLogger
}

func NewAgent(client client.Client, info serverVersionInfo, dataStore datastore.DataStore, log *zap.SugaredLogger) agent.Agent {
	return kubernetesAgent{
		Client:            client,
		serverVersionInfo: info,
		dataStore:         dataStore,
		log:               log,
	}
}

// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch

func (a kubernetesAgent) Collect(ctx context.Context) error {
	serverVersion, err := a.ServerVersion()
	if err != nil {
		return err
	}

	record := Record{
		KindVersion: agent.KindVersion{
			Kind:    "kubernetes",
			Version: telemetryversion.V1Version,
		},
		Time:              time.Now().UTC(),
		KubernetesVersion: serverVersion.String(),
	}

	knodes := &corev1.NodeList{}
	if err := a.List(ctx, knodes); err != nil {
		return err
	}
	for _, knode := range knodes.Items {
		node, err := nodeFromKubeNode(knode)
		if err != nil {
			return err
		}
		record.Nodes = append(record.Nodes, node)
	}

	a.log.Infow("Collected nodes", "nodes", len(record.Nodes))

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return a.dataStore.Store(ctx, data)
}

func nodeFromKubeNode(kn corev1.Node) (Node, error) {
	id, err := getID(kn)
	if err != nil {
		return Node{}, err
	}
	n := Node{
		ID:                      id,
		OperatingSystem:         agent.StrPtr(kn.Status.NodeInfo.OperatingSystem),
		OSImage:                 agent.StrPtr(kn.Status.NodeInfo.OSImage),
		KernelVersion:           agent.StrPtr(kn.Status.NodeInfo.KernelVersion),
		Architecture:            agent.StrPtr(kn.Status.NodeInfo.Architecture),
		ContainerRuntimeVersion: agent.StrPtr(kn.Status.NodeInfo.ContainerRuntimeVersion),
		KubeletVersion:          agent.StrPtr(kn.Status.NodeInfo.KubeletVersion),
		CloudProvider:           agent.StrPtr(kubernetes.ProviderName(kn.Spec.ProviderID)),
	}
	// We want to iterate the resources in a deterministic order.
	var keys []string
	for k := range kn.Status.Capacity {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := kn.Status.Capacity[corev1.ResourceName(k)]
		n.Capacity = append(n.Capacity, Resource{
			Resource: k,
			Value:    v.String(),
		})
	}
	return n, nil
}

func getID(kn corev1.Node) (string, error) {
	// We don't want to report the node's Name - that is Personally Identifiable Information.
	// The MachineID is apparently not always populated and SystemUUID is ill-defined. Let's
	// just hash them all together. It should be stable, and this reduces risk
	// of PII leakage.
	return agent.HashOf(kn.Name + kn.Status.NodeInfo.MachineID + kn.Status.NodeInfo.SystemUUID)
}

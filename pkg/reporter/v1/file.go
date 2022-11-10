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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kubermatic/telemetry-client/pkg/datastore"
	v1 "github.com/kubermatic/telemetry-client/pkg/report/v1"
	"github.com/kubermatic/telemetry-client/pkg/reporter"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

type fileReporter struct {
	dataStore  datastore.DataStore
	path       string
	clientUUID string
}

func NewFileReporter(dataStore datastore.DataStore, path, clientUUID string) (reporter.Reporter, error) {
	_, err := os.Stat(path)
	if err != nil {
		return fileReporter{}, err
	}
	return fileReporter{dataStore: dataStore, path: path, clientUUID: clientUUID}, nil
}

func (d fileReporter) Report(ctx context.Context) error {
	k8sClient, err := getClient()
	if err != nil {
		return err
	}

	ip, err := getNodeIP(ctx, k8sClient)
	if err != nil {
		return err
	}

	info, err := os.Stat(d.path)
	if err != nil {
		return err
	}

	files := []string{}
	if info.IsDir() {
		entries, err := os.ReadDir(d.path)
		if err != nil {
			return err
		}

		for _, e := range entries {
			files = append(files, e.Name())
		}
	} else {
		files = append(files, info.Name())
	}

	report := &v1.Report{
		Version:    "v1",
		Time:       time.Now().UTC(),
		ClientUUID: d.clientUUID,
		MasterIP:   ip,
	}

	for _, file := range files {
		b, err := os.ReadFile(filepath.Join(d.path, file))
		if err != nil {
			return err
		}

		report.Records = append(report.Records, b)
	}

	data, err := json.Marshal(report)
	if err != nil {
		return err
	}

	return d.dataStore.Store(ctx, data)
}

func getClient() (client.Client, error) {
	cfg := ctrl.GetConfigOrDie()
	mapper, err := apiutil.NewDynamicRESTMapper(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating rest mapper: %w", err)
	}
	c, err := client.New(cfg, client.Options{
		Scheme: scheme,
		Mapper: mapper,
	})
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	return c, nil
}

func getNodeIP(ctx context.Context, k8sClient client.Client) (string, error) {
	nodeList := &corev1.NodeList{}
	if err := k8sClient.List(ctx,
		nodeList,
		&client.ListOptions{Limit: 1}); err != nil {
		return "", fmt.Errorf("failed to list nodes: %w", err)
	}
	for _, node := range nodeList.Items {
		return getNodeExternalIP(node), nil
	}
	return "", nil
}

func getNodeExternalIP(node corev1.Node) string {
	for _, nodeAddress := range node.Status.Addresses {
		if nodeAddress.Type == corev1.NodeExternalIP {
			return nodeAddress.Address
		}
	}

	return ""
}

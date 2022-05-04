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

package kubermaticagent

import (
	"context"
	"fmt"

	k8cv1 "github.com/kubermatic/telemetry-client/pkg/agent/kubermatic/v1"
	"github.com/kubermatic/telemetry-client/pkg/datastore"

	"github.com/spf13/cobra"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(kubermaticv1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

type flags struct {
	// recordDir is the directory to save all records files from agents.
	recordDir string
}

func NewKubermaticAgentCommand() *cobra.Command {
	flags := &flags{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "kubermatic-agent",
		Short: "Kubermatic Telemetry Agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), flags)
		},
	}
	cmd.Flags().StringVar(&flags.recordDir, "record-dir", "/records/", "the directory to save all records files from agents")
	return cmd
}

func runE(ctx context.Context, flags *flags) error {
	cfg := ctrl.GetConfigOrDie()
	mapper, err := apiutil.NewDynamicRESTMapper(cfg)
	if err != nil {
		return fmt.Errorf("failed to create rest mapper: %w", err)
	}

	c, err := client.New(cfg, client.Options{
		Scheme: scheme,
		Mapper: mapper,
	})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	dataStore := datastore.NewFileStore(flags.recordDir)
	agent := k8cv1.NewAgent(c, discoveryClient, dataStore)

	return agent.Collect(ctx)
}

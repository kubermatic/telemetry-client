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

package reporter

import (
	"os"

	"github.com/kubermatic/telemetry-client/pkg/datastore"
	reporterv2 "github.com/kubermatic/telemetry-client/pkg/reporter/v2"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

type httpFlags struct {
	url string
	// recordDir is the directory for reporter to read reports.
	recordDir string
	// clientUUID is the clientUUID of this reporter.
	clientUUID string
}

func newHTTPReporterCommand(log *zap.SugaredLogger) *cobra.Command {
	flags := &httpFlags{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "http",
		Short: "Telemetry http-reporter",
		RunE: func(cmd *cobra.Command, args []string) error {
			httpStore := datastore.NewHTTPStore(flags.url, log)
			reporter, err := reporterv2.NewFileReporter(httpStore, flags.recordDir, flags.clientUUID)
			if err != nil {
				return err
			}
			return reporter.Report(cmd.Context())
		},
	}
	cmd.Flags().StringVar(&flags.recordDir, "record-dir", "/records/", "the directory for reporter to read reports")
	cmd.Flags().StringVar(&flags.url, "url", "", "the URL to push reports to")
	cmd.Flags().StringVar(&flags.clientUUID, "client-uuid", os.Getenv("CLIENT_UUID"), "the client UUID of this reporter")
	return cmd
}

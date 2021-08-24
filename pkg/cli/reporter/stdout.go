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

	"github.com/spf13/cobra"

	"github.com/kubermatic/telemetry-client/pkg/datastore"
	reporterv1 "github.com/kubermatic/telemetry-client/pkg/reporter/v1"
)

type stdoutFlags struct {
	// recordDir is the directory for reporter to read reports.
	recordDir string
	// clientUUID is the clientUUID of this reporter.
	clientUUID string
}

func newStdoutReporterCommand() *cobra.Command {
	flags := &stdoutFlags{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "stdout",
		Short: "Telemetry stdout-reporter",
		RunE: func(cmd *cobra.Command, args []string) error {
			stdoutStore := datastore.NewStdout()
			reporter, err := reporterv1.NewFileReporter(stdoutStore, flags.recordDir, flags.clientUUID)
			if err != nil {
				return err
			}
			return reporter.Report()

		},
	}
	cmd.Flags().StringVar(&flags.recordDir, "record-dir", "/records/", "the directory for reporter to read reports.")
	cmd.Flags().StringVar(&flags.clientUUID, "client-uuid", os.Getenv("CLIENT_UUID"), "the client uuid of this reporter.")
	return cmd
}

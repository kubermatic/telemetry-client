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

package main

import (
	"os"

	kubermaticagent "github.com/kubermatic/telemetry-client/pkg/cli/kubermatic-agent"
	"github.com/kubermatic/telemetry-client/pkg/log"
)

func main() {
	logger := log.NewDefault().Sugar().With("agent", "kubermatic")

	if err := kubermaticagent.NewKubermaticAgentCommand(logger).Execute(); err != nil {
		logger.Error(err)
		os.Exit(1)
	}

	logger.Info("Operation completed.")
}

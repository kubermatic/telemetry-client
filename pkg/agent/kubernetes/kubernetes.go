/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Telemetry takes the providerName method from Kubernetes spartakus project:
https://github.com/kubernetes-retired/spartakus/blob/master/pkg/volunteer/kubernetes.go#L118-L124
Telemetry modifies it to return directly if the string matches <ProviderName>://<ProviderSpecficNodeID> format.
*/

package kubernetes

import (
	"strings"
)

// ProviderName extracts the cloud provider name from a given
// string that should match: <ProviderName>://<ProviderSpecficNodeID>
// If the given string does not match this format, we return "unknown".
func ProviderName(providerID string) string {
	parts := strings.Split(providerID, "://")
	if len(parts) == 2 {
		return parts[0]
	}
	return "unknown"
}

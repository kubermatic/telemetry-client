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
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubermatic/telemetry-client/pkg/report"
)

type Report struct {
	Version    string            `json:"version"`
	Time       time.Time         `json:"time"`
	ClientUUID string            `json:"client_uuid"`
	Location   report.Location   `json:"location,omitempty"`
	Records    []json.RawMessage `json:"records,omitempty"`
}

func (r *Report) String() string {
	return fmt.Sprintf("Report version: %s, time: %v, clientUUID: %s", r.Version, r.Time, r.ClientUUID)
}

func (r *Report) ListRecords() []json.RawMessage {
	return r.Records
}

func (r *Report) SetLocation(location report.Location) {
	r.Location = location
}

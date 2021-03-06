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

package datastore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

type stdout struct {
}

func NewStdout() DataStore {
	return stdout{}
}

func (f stdout) Store(ctx context.Context, data json.RawMessage) error {
	var j bytes.Buffer
	if err := json.Indent(&j, data, "", "\t"); err != nil {
		return err
	}
	fmt.Println(j.String())
	return nil
}

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
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/kubermatic/telemetry-client/pkg/datastore"
	v1 "github.com/kubermatic/telemetry-client/pkg/report/v1"
	"github.com/kubermatic/telemetry-client/pkg/reporter"
)

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

func (d fileReporter) Report() error {
	report := &v1.Report{
		Version:    "v1",
		Time:       time.Now().UTC(),
		ClientUUID: d.clientUUID,
	}
	info, err := os.Stat(d.path)
	if err != nil {
		return err
	}
	var files []os.FileInfo
	if info.IsDir() {
		if files, err = ioutil.ReadDir(d.path); err != nil {
			return err
		}
	} else {
		files = []os.FileInfo{info}
	}

	for _, file := range files {
		b, err := ioutil.ReadFile(filepath.Join(d.path, file.Name()))
		if err != nil {
			return err
		}
		report.Records = append(report.Records, b)
	}
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	return d.dataStore.Store(data)
}

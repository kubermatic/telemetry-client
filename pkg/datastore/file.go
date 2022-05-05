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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/rand"
)

type fileStore struct {
	directory string
	log       *zap.SugaredLogger
}

func NewFileStore(directory string, log *zap.SugaredLogger) DataStore {
	return fileStore{directory: directory, log: log}
}

func (s fileStore) Store(ctx context.Context, data json.RawMessage) error {
	info, err := os.Stat(s.directory)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return err
	}

	now := time.Now().UTC().Format("2006-01-02T15-04-05")
	filename := filepath.Join(s.directory, fmt.Sprintf("record-%s-%s.json", now, rand.String(6)))

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}

	s.log.Infow("Stored data on disk", "filename", filename)

	return nil
}

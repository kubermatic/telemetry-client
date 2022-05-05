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
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
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

	// We need to provide a different Seed every time when generating a new file, otherwise, it will use the default Seed,
	// which will provide a deterministic file name every time, and this will cause data overwriting.
	rand.Seed(time.Now().UnixNano())
	filename := filepath.Join(s.directory, fmt.Sprintf("record-%s.json", fmt.Sprint(rand.Uint64())))

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

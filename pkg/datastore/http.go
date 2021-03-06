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
	"net/http"

	"go.uber.org/zap"
)

type httpStore struct {
	url string
	log *zap.SugaredLogger
}

func NewHTTPStore(endpoint string, log *zap.SugaredLogger) DataStore {
	return httpStore{url: endpoint, log: log}
}

func (s httpStore) Store(ctx context.Context, data json.RawMessage) error {
	req, err := http.NewRequestWithContext(ctx, "POST", s.url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	s.log.Infow("Sending data via HTTP…", "target", s.url)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

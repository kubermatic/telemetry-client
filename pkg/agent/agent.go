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

package agent

import (
	"context"
	"crypto/md5"
	"encoding/hex"
)

type Agent interface {
	Collect(ctx context.Context) error
}

func HashOf(str string) (string, error) {
	hasher := md5.New()
	_, err := hasher.Write([]byte(str))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)[0:]), nil
}

func StrPtr(str string) *string {
	if str == "" {
		return nil
	}

	return &str
}

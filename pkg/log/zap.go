/*
Copyright 2022 The Telemetry Authors.

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

package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlruntimelzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Format string

const (
	FormatJSON    Format = "JSON"
	FormatConsole Format = "Console"
)

func New(debug bool, format Format) *zap.Logger {
	// this basically mimics New<type>Config, but with a custom sink
	sink := zapcore.AddSync(os.Stderr)

	// Level - We only support setting Info+ or Debug+
	lvl := zap.NewAtomicLevelAt(zap.InfoLevel)
	if debug {
		lvl = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	encCfg := zap.NewProductionEncoderConfig()
	// Having a dateformat makes it more easy to look at logs outside of something like Kibana
	encCfg.TimeKey = "time"
	encCfg.EncodeTime = zapcore.RFC3339TimeEncoder

	// production config encodes durations as a float of the seconds value, but we want a more
	// readable, precise representation
	encCfg.EncodeDuration = zapcore.StringDurationEncoder

	var enc zapcore.Encoder
	if format == FormatJSON {
		enc = zapcore.NewJSONEncoder(encCfg)
	} else if format == FormatConsole {
		enc = zapcore.NewConsoleEncoder(encCfg)
	}

	opts := []zap.Option{
		zap.AddCaller(),
		zap.ErrorOutput(sink),
	}

	coreLog := zapcore.NewCore(&ctrlruntimelzap.KubeAwareEncoder{Encoder: enc}, sink, lvl)
	return zap.New(coreLog, opts...)
}

// NewDefault creates new default logger.
func NewDefault() *zap.Logger {
	return New(false, FormatJSON)
}

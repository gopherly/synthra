// Copyright 2026 The Gopherly Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gopherly.dev/synthra"
)

func TestCustomValidator_AcceptsValidTLS(t *testing.T) {
	cfg, err := synthra.New(
		synthra.WithFile("config.yaml"),
		synthra.WithValidator(tlsPathsConsistent),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
}

func TestCustomValidator_RejectsIncompleteTLS(t *testing.T) {
	cfg, err := synthra.New(
		synthra.WithFile("config-invalid.yaml"),
		synthra.WithValidator(tlsPathsConsistent),
	)
	require.NoError(t, err)
	require.Error(t, cfg.Load(context.Background()))
}

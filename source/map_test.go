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

//go:build !integration

package source

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMap_nilReturnsEmpty(t *testing.T) {
	t.Parallel()
	m := NewMap(nil)
	got, err := m.Load(context.Background())
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)
}

func TestNewMap_returnsSameReference(t *testing.T) {
	t.Parallel()
	data := map[string]any{"k": "v"}
	m := NewMap(data)
	got1, err1 := m.Load(context.Background())
	require.NoError(t, err1)
	got2, err2 := m.Load(context.Background())
	require.NoError(t, err2)
	require.Equal(t, data, got1)
	require.Equal(t, got1, got2)
}

func TestNewMap_concurrentLoad_readOnly(t *testing.T) {
	t.Parallel()
	m := NewMap(map[string]any{"x": 1})
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			got, err := m.Load(context.Background())
			require.NoError(t, err)
			require.Equal(t, 1, got["x"])
		})
	}
	wg.Wait()
}

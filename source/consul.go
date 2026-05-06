// Copyright 2026 The Gopherly Authors
// Copyright 2025 Company.info B.V.
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

package source

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul/api"
	"gopherly.dev/synthra/codec"
)

// ConsulKV defines the interface for Consul key-value operations.
// This interface enables testing by allowing mock implementations.
type ConsulKV interface {
	Get(key string, q *api.QueryOptions) (*api.KVPair, *api.QueryMeta, error)
}

// Consul represents a configuration source that loads data from Consul's
// key-value store.
//
// The Consul client is configured using environment variables:
//   - CONSUL_HTTP_ADDR: The address of the Consul server
//     (e.g., "http://localhost:8500")
//   - CONSUL_HTTP_TOKEN: The access token for authentication (optional)
type Consul struct {
	client    *api.Client
	kv        ConsulKV
	path      string
	lastIndex uint64
	decoder   codec.Decoder
}

// NewConsul creates a new Consul configuration source with the given path
// and decoder.
// The path parameter specifies the key path in Consul's key-value store.
// If kv is nil, it uses the default Consul client KV implementation.
//
// Errors:
//   - Returns error if the Consul client cannot be created
func NewConsul(path string, decoder codec.Decoder, kv ConsulKV) (*Consul, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}
	if kv == nil {
		kv = client.KV()
	}
	return &Consul{
		client:  client,
		kv:      kv,
		path:    path,
		decoder: decoder,
	}, nil
}

// Load retrieves configuration data from the Consul key-value store at
// the configured path.
// If the key does not exist in Consul, it returns an empty map without error.
//
// Errors:
//   - Returns error if the Consul query fails
//   - Returns error if decoding the value fails
func (c *Consul) Load(ctx context.Context) (map[string]any, error) {
	pair, meta, err := c.kv.Get(c.path, (&api.QueryOptions{}).WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get consul key: %w", err)
	}

	if pair == nil {
		return make(map[string]any), nil
	}

	if meta != nil {
		c.lastIndex = meta.LastIndex
	}

	var config map[string]any
	if err = c.decoder.Decode(pair.Value, &config); err != nil {
		return nil, fmt.Errorf("failed to decode consul value: %w", err)
	}

	return config, nil
}

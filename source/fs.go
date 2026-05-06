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

package source

import (
	"context"
	"fmt"
	"io/fs"

	"gopherly.dev/synthra/codec"
)

// FileFS loads configuration from a single file inside an [io/fs.FS].
// Concurrent use is OK when the underlying [fs.FS] is safe for concurrent use.
type FileFS struct {
	fsys    fs.FS
	name    string
	decoder codec.Decoder
}

// NewFileFS returns a source that reads name from fsys and decodes bytes with
// decoder.
// name must use slash-separated paths as required by [fs.FS]
// (for example "config/app.yaml").
func NewFileFS(fsys fs.FS, name string, decoder codec.Decoder) *FileFS {
	return &FileFS{fsys: fsys, name: name, decoder: decoder}
}

// Load reads the named file from fsys and decodes it into a map[string]any.
// Keys in the returned map are not normalized; the Synthra loader normalizes keys.
func (f *FileFS) Load(_ context.Context) (map[string]any, error) {
	if f.fsys == nil {
		return nil, fmt.Errorf("file fs source: filesystem is nil")
	}
	if f.decoder == nil {
		return nil, fmt.Errorf("file fs source: decoder is nil")
	}

	data, err := fs.ReadFile(f.fsys, f.name)
	if err != nil {
		return nil, fmt.Errorf("file fs source: read %q: %w", f.name, err)
	}

	var config map[string]any
	if err = f.decoder.Decode(data, &config); err != nil {
		return nil, fmt.Errorf("file fs source: decode %q: %w", f.name, err)
	}

	return config, nil
}

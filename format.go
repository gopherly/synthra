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

package synthra

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopherly.dev/synthra/codec"
)

// extensionFormats maps file extensions to codecs for automatic format detection.
var extensionFormats = map[string]codec.Codec{
	".yaml": codec.YAML,
	".yml":  codec.YAML,
	".json": codec.JSON,
	".toml": codec.TOML,
}

// detectFormat automatically detects the codec based on the file extension.
// It returns an error if the format cannot be determined from the extension.
func detectFormat(path string) (codec.Codec, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if c, ok := extensionFormats[ext]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("cannot detect format from extension %q; use WithFileAs() to specify format explicitly", ext)
}

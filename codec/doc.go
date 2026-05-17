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

// Package codec provides encoding and decoding functionality for
// configuration data.
//
// The codec package defines [Encoder] and [Decoder] interfaces for converting
// configuration data between different formats (JSON, YAML, TOML, etc.) and
// Go types. Built-in codecs are package-level values such as [JSON], [YAML],
// and [TOML].
//
// # Built-in Codecs
//
// The package includes built-in support for common formats:
//
//   - JSON: Standard JSON encoding/decoding
//   - YAML: YAML encoding/decoding
//   - TOML: TOML encoding/decoding
//   - EnvVar: Environment variable format
//
// # Custom Codecs
//
// Implement [Codec] (or [Decoder] alone when you only load) and pass the value
// to [gopherly.dev/synthra.WithFileAs], [gopherly.dev/synthra.WithFileFS],
// [gopherly.dev/synthra.WithFileFSAs], [gopherly.dev/synthra.WithContent],
// [gopherly.dev/synthra/source.NewFile], or
// [gopherly.dev/synthra/source.NewFileFS] as appropriate.
//
// Example with an explicit decoder and no file extension:
//
//	cfg, err := synthra.New(
//	    synthra.WithFileAs("settings.dat", myCodec),
//	)
//
// where myCodec implements [Codec].
//
// # Scalar Decoders
//
// The package includes scalar decoders for parsing individual values:
//
//	decoder := codec.ParseInt("port")
//	var m map[string]any
//	decoder.Decode([]byte("8080"), &m)  // m["port"] is int(8080)
//
// Supported scalar parsers include ParseBool, ParseString, ParseInt variants,
// ParseUint variants, ParseFloat variants, ParseDuration, ParseTime, and ParseAs
// for custom types.
package codec

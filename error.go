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

package synthra

import (
	"errors"
	"fmt"
)

// Op values identify which Synthra entrypoint produced a [ConfigError].
// They follow the lowercase convention used by [os.PathError.Op],
// [net.OpError.Op], and [net/url.Error.Op].
const (
	OpNew  = "new"
	OpLoad = "load"
	OpDump = "dump"
	OpGet  = "get"
)

// ErrNilConfig is returned when a typed accessor or [Get] is used on
// a nil [*Synthra].
var ErrNilConfig = errors.New("synthra: nil Synthra")

// ErrKeyNotFound is returned when a configuration key is missing or cannot be
// resolved for strict accessors. Errors may wrap this value; use [errors.Is]
// to detect it.
var ErrKeyNotFound = errors.New("synthra: key not found")

// ErrNilContext is returned when [Synthra.Load] or [Synthra.Dump] is called
// with a nil [context.Context].
var ErrNilContext = errors.New("synthra: nil context")

// ConfigError is the structured error returned by Synthra for construction,
// load, dump, and type conversion failures at accessors.
//
// Its shape follows [os.PathError]: Op names the operation, Path locates the
// failure in a way that depends on Op (see package docs), and Err is the
// underlying cause for [errors.Unwrap], [errors.Is], and [errors.As].
//
// Path is diagnostic text only; its format is not a stable API contract.
// Callers should branch on Op and use [errors.Is] on Err for specific reasons,
// not parse Path or [ConfigError.Error] output.
type ConfigError struct {
	Op   string
	Path string
	Err  error
}

// Error implements [error]. The format is pinned for tests:
//
//	synthra: <Op>[ <Path>]: <Err>
//
// When Path is empty, the space before Path is omitted.
func (e *ConfigError) Error() string {
	if e == nil {
		return "synthra: <nil>"
	}
	if e.Path != "" {
		if e.Err != nil {
			return fmt.Sprintf("synthra: %s %s: %v", e.Op, e.Path, e.Err)
		}
		return fmt.Sprintf("synthra: %s %s", e.Op, e.Path)
	}
	if e.Err != nil {
		return fmt.Sprintf("synthra: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("synthra: %s", e.Op)
}

// Unwrap returns the underlying error for [errors.Is] and [errors.As].
func (e *ConfigError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewConfigError returns a [*ConfigError]. Op should be one of [OpNew],
// [OpLoad], [OpDump], or [OpGet]. Path is the polymorphic locator described on
// [ConfigError]; use "" when none applies (for example nil-context errors).
func NewConfigError(op, path string, err error) *ConfigError {
	return &ConfigError{Op: op, Path: path, Err: err}
}

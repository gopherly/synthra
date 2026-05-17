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
	"reflect"
	"time"

	"github.com/spf13/cast"
)

// Get returns the value associated with the given key as type T.
// If c is nil, it returns [ErrNilConfig].
// If the key is missing or empty, or the value cannot be converted to T,
// it returns an error.
//
// Example:
//
//	port, err := synthra.Get[int](cfg, "server.port")
//	if err != nil {
//	    return fmt.Errorf("server.port: %w", err)
//	}
//
//	timeout, err := synthra.Get[time.Duration](cfg, "timeout")
//	if err != nil {
//	    return err
//	}
func Get[T any](c *Synthra, key string) (T, error) {
	zero := getZeroValue[T]()
	val, err := c.requireValue(key)
	if err != nil {
		return zero, err
	}

	if result, ok := val.(T); ok {
		return result, nil
	}

	result, ok := convertToType[T](val)
	if ok {
		return result, nil
	}

	return zero, NewConfigError(OpGet, key, fmt.Errorf("cannot convert to %T", zero))
}

// GetOr returns the value associated with the given key as type T.
// If the key is not found or cannot be converted to type T, it returns the
// provided default value.
// The type T is inferred from the default value.
//
// Example:
//
//	port := synthra.GetOr(cfg, "server.port", 8080)           // type inferred as int
//	host := synthra.GetOr(cfg, "server.host", "localhost")    // type inferred as string
//	timeout := synthra.GetOr(cfg, "timeout", 30*time.Second)  // type inferred as time.Duration
func GetOr[T any](c *Synthra, key string, defaultVal T) T {
	if c == nil {
		return defaultVal
	}

	val := c.getValueFromMap(key)
	if val == nil {
		return defaultVal
	}

	// Try direct type assertion first
	if result, ok := val.(T); ok {
		return result
	}

	// Fallback to cast library for common type conversions
	result, ok := convertToType[T](val)
	if ok {
		return result
	}

	return defaultVal
}

// String returns the value at key as a string.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	host, err := cfg.String("server.host")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) String(key string) (string, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return "", err
	}
	s, err := cast.ToStringE(v)
	if err != nil {
		return "", NewConfigError(OpGet, key, err)
	}
	return s, nil
}

// Int returns the value at key as an int.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	port, err := cfg.Int("server.port")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) Int(key string) (int, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return 0, err
	}
	i, err := cast.ToIntE(v)
	if err != nil {
		return 0, NewConfigError(OpGet, key, err)
	}
	return i, nil
}

// Int64 returns the value at key as an int64.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	maxSize, err := cfg.Int64("max_size")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) Int64(key string) (int64, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return 0, err
	}
	i, err := cast.ToInt64E(v)
	if err != nil {
		return 0, NewConfigError(OpGet, key, err)
	}
	return i, nil
}

// Float64 returns the value at key as a float64.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	rate, err := cfg.Float64("rate")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) Float64(key string) (float64, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return 0, err
	}
	f, err := cast.ToFloat64E(v)
	if err != nil {
		return 0, NewConfigError(OpGet, key, err)
	}
	return f, nil
}

// Bool returns the value at key as a bool.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	debug, err := cfg.Bool("debug")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) Bool(key string) (bool, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return false, err
	}
	b, err := cast.ToBoolE(v)
	if err != nil {
		return false, NewConfigError(OpGet, key, err)
	}
	return b, nil
}

// Duration returns the value at key as a [time.Duration].
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	timeout, err := cfg.Duration("timeout")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) Duration(key string) (time.Duration, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return 0, err
	}
	d, err := cast.ToDurationE(v)
	if err != nil {
		return 0, NewConfigError(OpGet, key, err)
	}
	return d, nil
}

// Time returns the value at key as a [time.Time].
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	startTime, err := cfg.Time("start_time")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) Time(key string) (time.Time, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return time.Time{}, err
	}
	tm, err := cast.ToTimeE(v)
	if err != nil {
		return time.Time{}, NewConfigError(OpGet, key, err)
	}
	return tm, nil
}

// StringSlice returns the value at key as a []string.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	tags, err := cfg.StringSlice("tags")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) StringSlice(key string) ([]string, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return nil, err
	}
	s, err := cast.ToStringSliceE(v)
	if err != nil {
		return nil, NewConfigError(OpGet, key, err)
	}
	return s, nil
}

// IntSlice returns the value at key as a []int.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	ports, err := cfg.IntSlice("ports")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) IntSlice(key string) ([]int, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return nil, err
	}
	s, err := cast.ToIntSliceE(v)
	if err != nil {
		return nil, NewConfigError(OpGet, key, err)
	}
	return s, nil
}

// StringMap returns the value at key as a map[string]any.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	metadata, err := cfg.StringMap("metadata")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) StringMap(key string) (map[string]any, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return nil, err
	}
	m, err := cast.ToStringMapE(v)
	if err != nil {
		return nil, NewConfigError(OpGet, key, err)
	}
	return m, nil
}

// StringOr returns the value associated with the given key as a string,
// or the default value if not found.
//
// Example:
//
//	host := cfg.StringOr("server.host", "localhost")
func (c *Synthra) StringOr(key, defaultVal string) string {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToString(val)
}

// IntOr returns the value associated with the given key as an int, or
// the default value if not found.
//
// Example:
//
//	port := cfg.IntOr("server.port", 8080)
func (c *Synthra) IntOr(key string, defaultVal int) int {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToInt(val)
}

// Int64Or returns the value associated with the given key as an int64,
// or the default value if not found.
//
// Example:
//
//	maxSize := cfg.Int64Or("max_size", 1024)
func (c *Synthra) Int64Or(key string, defaultVal int64) int64 {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToInt64(val)
}

// Float64Or returns the value associated with the given key as a float64,
// or the default value if not found.
//
// Example:
//
//	rate := cfg.Float64Or("rate", 0.5)
func (c *Synthra) Float64Or(key string, defaultVal float64) float64 {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToFloat64(val)
}

// BoolOr returns the value associated with the given key as a boolean,
// or the default value if not found.
//
// Example:
//
//	debug := cfg.BoolOr("debug", false)
func (c *Synthra) BoolOr(key string, defaultVal bool) bool {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToBool(val)
}

// DurationOr returns the value associated with the given key as a
// [time.Duration], or the default value if not found.
//
// Example:
//
//	timeout := cfg.DurationOr("timeout", 30*time.Second)
func (c *Synthra) DurationOr(key string, defaultVal time.Duration) time.Duration {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToDuration(val)
}

// TimeOr returns the value associated with the given key as a [time.Time],
// or the default value if not found.
//
// Example:
//
//	startTime := cfg.TimeOr("start_time", time.Now())
func (c *Synthra) TimeOr(key string, defaultVal time.Time) time.Time {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToTime(val)
}

// StringSliceOr returns the value associated with the given key as a
// slice of strings, or the default value if not found.
//
// Example:
//
//	tags := cfg.StringSliceOr("tags", []string{"default"})
func (c *Synthra) StringSliceOr(key string, defaultVal []string) []string {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToStringSlice(val)
}

// IntSliceOr returns the value associated with the given key as a slice
// of integers, or the default value if not found.
//
// Example:
//
//	ports := cfg.IntSliceOr("ports", []int{8080, 8081})
func (c *Synthra) IntSliceOr(key string, defaultVal []int) []int {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToIntSlice(val)
}

// StringMapOr returns the value associated with the given key as a
// map[string]any, or the default value if not found.
//
// Example:
//
//	metadata := cfg.StringMapOr("metadata", map[string]any{"version": "1.0"})
func (c *Synthra) StringMapOr(key string, defaultVal map[string]any) map[string]any {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToStringMap(val)
}

// getZeroValue returns a proper zero value for type T.
// For slices and maps, it returns empty initialized values instead of nil.
func getZeroValue[T any]() T {
	var zero T
	v := reflect.ValueOf(&zero).Elem()

	// Initialize slices and maps to empty instead of nil
	switch v.Kind() {
	case reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	case reflect.Map:
		v.Set(reflect.MakeMap(v.Type()))
	}

	return zero
}

// convertToType attempts to convert a value to type T using the cast library.
// This handles common type conversions (int, string, bool, etc.) but won't
// work for custom types.
func convertToType[T any](val any) (T, bool) {
	var zero T
	var result any

	// Use type switch to handle common conversions
	switch any(zero).(type) {
	case string:
		result = cast.ToString(val)
	case int:
		result = cast.ToInt(val)
	case int64:
		result = cast.ToInt64(val)
	case int32:
		result = cast.ToInt32(val)
	case int16:
		result = cast.ToInt16(val)
	case int8:
		result = cast.ToInt8(val)
	case uint:
		result = cast.ToUint(val)
	case uint64:
		result = cast.ToUint64(val)
	case uint32:
		result = cast.ToUint32(val)
	case uint16:
		result = cast.ToUint16(val)
	case uint8:
		result = cast.ToUint8(val)
	case float64:
		result = cast.ToFloat64(val)
	case float32:
		result = cast.ToFloat32(val)
	case bool:
		result = cast.ToBool(val)
	case []string:
		result = cast.ToStringSlice(val)
	case []int:
		result = cast.ToIntSlice(val)
	case map[string]any:
		result = cast.ToStringMap(val)
	case map[string]string:
		result = cast.ToStringMapString(val)
	case map[string][]string:
		result = cast.ToStringMapStringSlice(val)
	case time.Duration:
		result = cast.ToDuration(val)
	case time.Time:
		result = cast.ToTime(val)
	default:
		return zero, false
	}

	// Convert result back to T
	if typedResult, ok := result.(T); ok {
		return typedResult, true
	}

	return zero, false
}

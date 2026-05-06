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

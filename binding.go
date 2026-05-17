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

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cast"
)

// Validator is an interface for structs that can validate their own configuration.
// The validation package uses the same contract (validation.Validator); a
// type implementing either satisfies both.
type Validator interface {
	Validate() error
}

// applyDefaults applies default values from struct tags to a struct.
// It walks through the struct fields and sets defaults for fields that
// have the 'default' tag and are currently zero-valued.
func applyDefaults(target any) error {
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Pointer {
		return fmt.Errorf("target must be a pointer")
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to a struct")
	}

	return setDefaults(val)
}

// setDefaults recursively sets default values on a struct.
func setDefaults(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			if err := setDefaults(field); err != nil {
				return err
			}
			continue
		}

		// Check if field has a default tag
		defaultTag := fieldType.Tag.Get("default")
		if defaultTag == "" {
			continue
		}

		// Only set default if field is zero-valued
		if !isZeroValue(field) {
			continue
		}

		// Set the default value based on field type
		if err := setDefaultValue(field, defaultTag); err != nil {
			return fmt.Errorf("failed to set default for field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// isZeroValue checks if a [reflect.Value] is the zero value for its type.
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	default:
		return false
	}
}

// setDefaultValue sets a default value on a field based on its type.
func setDefaultValue(field reflect.Value, defaultVal string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultVal)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Special handling for time.Duration
		if field.Type() == reflect.TypeFor[time.Duration]() {
			d, err := time.ParseDuration(defaultVal)
			if err != nil {
				return err
			}
			field.SetInt(int64(d))
		} else {
			i, err := cast.ToInt64E(defaultVal)
			if err != nil {
				return err
			}
			field.SetInt(i)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := cast.ToUint64E(defaultVal)
		if err != nil {
			return err
		}
		field.SetUint(u)
	case reflect.Float32, reflect.Float64:
		f, err := cast.ToFloat64E(defaultVal)
		if err != nil {
			return err
		}
		field.SetFloat(f)
	case reflect.Bool:
		b, err := cast.ToBoolE(defaultVal)
		if err != nil {
			return err
		}
		field.SetBool(b)
	default:
		return fmt.Errorf("unsupported type for default tag: %s", field.Kind())
	}
	return nil
}

// getDecoderConfig returns a cached decoder configuration to reduce
// reflection overhead.
func (c *Synthra) getDecoderConfig() *mapstructure.DecoderConfig {
	c.decoderOnce.Do(func() {
		tagName := c.tagName
		if tagName == "" {
			tagName = "synthra" // Fallback to default
		}
		c.decoderConfig = &mapstructure.DecoderConfig{
			TagName:          tagName,
			Squash:           true,
			WeaklyTypedInput: true,
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.StringToSliceHookFunc(","),
				mapstructure.StringToTimeHookFunc(time.RFC3339),
				mapstructure.StringToURLHookFunc(),
			),
		}
	})
	return c.decoderConfig
}

// decodeBindingInto decodes values into target using mapstructure. Errors
// match the messages produced by the former bind/bindAndValidate helpers.
func (c *Synthra) decodeBindingInto(target, values any) error {
	decoderCfg := c.getDecoderConfig()
	decoderCfg.Result = target

	decoder, err := mapstructure.NewDecoder(decoderCfg)
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	if err = decoder.Decode(values); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return nil
}

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
	"sync"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cast"
)

// defaultsCache caches whether a struct type has any `default` tags,
// avoiding repeated reflection scans on repeated Load calls.
var defaultsCache sync.Map // map[reflect.Type]bool

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

	if !structHasDefaults(typ) {
		return nil
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanSet() {
			continue
		}

		switch field.Kind() {
		case reflect.Struct:
			if err := setDefaults(field); err != nil {
				return err
			}
			continue
		case reflect.Pointer:
			if field.Type().Elem().Kind() == reflect.Struct {
				if field.IsNil() {
					if structHasDefaults(field.Type().Elem()) {
						field.Set(reflect.New(field.Type().Elem()))
					} else {
						continue
					}
				}
				if err := setDefaults(field.Elem()); err != nil {
					return err
				}
				continue
			}
		}

		defaultTag := fieldType.Tag.Get("default")
		if defaultTag == "" {
			continue
		}

		if !field.IsZero() {
			continue
		}

		if err := setDefaultValue(field, defaultTag); err != nil {
			return fmt.Errorf("failed to set default for field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// structHasDefaults reports whether typ (must be a struct type) or any of its
// nested structs carry at least one `default` tag. Results are cached.
func structHasDefaults(typ reflect.Type) bool {
	if v, loaded := defaultsCache.Load(typ); loaded {
		if has, ok := v.(bool); ok {
			return has
		}
	}
	has := scanDefaults(typ)
	defaultsCache.Store(typ, has)
	return has
}

func scanDefaults(typ reflect.Type) bool {
	for f := range typ.Fields() {
		if !f.IsExported() {
			continue
		}
		if f.Tag.Get("default") != "" {
			return true
		}
		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct && structHasDefaults(ft) {
			return true
		}
	}
	return false
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

// decodeBindingInto decodes values into target using mapstructure.
//
// A fresh DecoderConfig is built on every call because mapstructure documents
// that a config must not be reused after NewDecoder returns. The DecodeHook
// composition cost is negligible compared to the decode itself.
//
// mapstructure's default field-matching function is [strings.EqualFold], so
// it is already case-insensitive. Keys canonicalized by canonicalizeSchemaKeys
// (or preserved as-is when no schema is present) flow naturally into struct
// fields regardless of casing differences between the config source and the
// struct tag.
func (c *Synthra) decodeBindingInto(target, values any) error {
	tagName := c.tagName
	if tagName == "" {
		tagName = "synthra"
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:          tagName,
		Result:           target,
		Squash:           true,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.StringToTimeHookFunc(time.RFC3339),
			mapstructure.StringToURLHookFunc(),
		),
	})
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	if err = decoder.Decode(values); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return nil
}

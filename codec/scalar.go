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

package codec

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type parseAsDecoder[T any] struct {
	key string
	fn  func(string) (T, error)
}

func (d parseAsDecoder[T]) Decode(data []byte, v any) error {
	m, ok := v.(*map[string]any)
	if !ok {
		return fmt.Errorf("codec: expected *map[string]any, got %T", v)
	}
	val, err := d.fn(strings.TrimSpace(string(data)))
	if err != nil {
		return err
	}
	*m = map[string]any{d.key: val}
	return nil
}

// ParseAs decodes raw bytes using fn and stores the result under the key.
func ParseAs[T any](key string, fn func(string) (T, error)) Decoder {
	return parseAsDecoder[T]{key: key, fn: fn}
}

// ParseBool decodes a bool value and stores it under the key.
func ParseBool(key string) Decoder { return ParseAs(key, strconv.ParseBool) }

// ParseString decodes a string value and stores it under the key.
func ParseString(key string) Decoder {
	return ParseAs(key, func(s string) (string, error) { return s, nil })
}

// ParseDuration decodes a [time.Duration] value and stores it under the key.
func ParseDuration(key string) Decoder { return ParseAs(key, time.ParseDuration) }

// ParseInt decodes an int value and stores it under the key.
func ParseInt(key string) Decoder { return ParseAs(key, strconv.Atoi) }

// ParseFloat64 decodes a float64 value and stores it under the key.
func ParseFloat64(key string) Decoder {
	return ParseAs(key, func(s string) (float64, error) { return strconv.ParseFloat(s, 64) })
}

// ParseFloat32 decodes a float32 value and stores it under the key.
func ParseFloat32(key string) Decoder {
	return ParseAs(key, func(s string) (float32, error) {
		f, err := strconv.ParseFloat(s, 32)
		return float32(f), err
	})
}

// ParseInt64 decodes an int64 value and stores it under the key.
func ParseInt64(key string) Decoder {
	return ParseAs(key, func(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) })
}

// ParseInt32 decodes an int32 value and stores it under the key.
func ParseInt32(key string) Decoder {
	return ParseAs(key, func(s string) (int32, error) {
		i, err := strconv.ParseInt(s, 10, 32)
		return int32(i), err
	})
}

// ParseInt16 decodes an int16 value and stores it under the key.
func ParseInt16(key string) Decoder {
	return ParseAs(key, func(s string) (int16, error) {
		i, err := strconv.ParseInt(s, 10, 16)
		return int16(i), err
	})
}

// ParseInt8 decodes an int8 value and stores it under the key.
func ParseInt8(key string) Decoder {
	return ParseAs(key, func(s string) (int8, error) {
		i, err := strconv.ParseInt(s, 10, 8)
		return int8(i), err
	})
}

// ParseUint decodes an uint value and stores it under the key.
func ParseUint(key string) Decoder {
	return ParseAs(key, func(s string) (uint, error) {
		u, err := strconv.ParseUint(s, 10, 0)
		return uint(u), err
	})
}

// ParseUint64 decodes an uint64 value and stores it under the key.
func ParseUint64(key string) Decoder {
	return ParseAs(key, func(s string) (uint64, error) { return strconv.ParseUint(s, 10, 64) })
}

// ParseUint32 decodes an uint32 value and stores it under the key.
func ParseUint32(key string) Decoder {
	return ParseAs(key, func(s string) (uint32, error) {
		u, err := strconv.ParseUint(s, 10, 32)
		return uint32(u), err
	})
}

// ParseUint16 decodes an uint16 value and stores it under the key.
func ParseUint16(key string) Decoder {
	return ParseAs(key, func(s string) (uint16, error) {
		u, err := strconv.ParseUint(s, 10, 16)
		return uint16(u), err
	})
}

// ParseUint8 decodes an uint8 value and stores it under the key.
func ParseUint8(key string) Decoder {
	return ParseAs(key, func(s string) (uint8, error) {
		u, err := strconv.ParseUint(s, 10, 8)
		return uint8(u), err
	})
}

// ParseTime decodes a [time.Time] value (RFC3339 format) and stores
// it under the key.
func ParseTime(key string) Decoder {
	return ParseAs(key, func(s string) (time.Time, error) { return time.Parse(time.RFC3339, s) })
}

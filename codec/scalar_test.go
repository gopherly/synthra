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

//go:build !integration

package codec

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAs_CustomType(t *testing.T) {
	type Celsius float64
	d := ParseAs("temp", func(s string) (Celsius, error) {
		f, err := strconv.ParseFloat(s, 64)
		return Celsius(f), err
	})
	var m map[string]any
	err := d.Decode([]byte("36.6"), &m)
	require.NoError(t, err)
	c, ok := m["temp"].(Celsius)
	require.True(t, ok, "expected Celsius, got %T", m["temp"])
	assert.InDelta(t, 36.6, float64(c), 0.001)
}

func TestParseAs_StrconvDirectly(t *testing.T) {
	d := ParseAs("pid", strconv.Atoi)
	var m map[string]any
	err := d.Decode([]byte("1234"), &m)
	require.NoError(t, err)
	assert.Equal(t, 1234, m["pid"])
}

func TestParseAs_WrongVType(t *testing.T) {
	d := ParseInt("x")
	var notMap string
	err := d.Decode([]byte("42"), &notMap)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected *map[string]any")
}

func TestParseAs_InvalidValue(t *testing.T) {
	d := ParseInt("x")
	var m map[string]any
	err := d.Decode([]byte("notanint"), &m)
	require.Error(t, err)
}

func TestParseAs_TrimSpace(t *testing.T) {
	d := ParseInt("x")
	var m map[string]any
	err := d.Decode([]byte("  42  "), &m)
	require.NoError(t, err)
	assert.Equal(t, 42, m["x"])
}

func TestParseAs_KeyIsCorrect(t *testing.T) {
	d := ParseInt("mykey")
	var m map[string]any
	err := d.Decode([]byte("7"), &m)
	require.NoError(t, err)
	_, ok := m["mykey"]
	assert.True(t, ok, "expected key 'mykey' in result map")
	assert.Equal(t, 7, m["mykey"])
}

func TestParseBool(t *testing.T) {
	d := ParseBool("enabled")
	var m map[string]any
	err := d.Decode([]byte("true"), &m)
	require.NoError(t, err)
	assert.Equal(t, true, m["enabled"])
}

func TestParseBool_False(t *testing.T) {
	d := ParseBool("enabled")
	var m map[string]any
	err := d.Decode([]byte("false"), &m)
	require.NoError(t, err)
	assert.Equal(t, false, m["enabled"])
}

func TestParseInt(t *testing.T) {
	d := ParseInt("count")
	var m map[string]any
	err := d.Decode([]byte("42"), &m)
	require.NoError(t, err)
	assert.Equal(t, 42, m["count"])
}

func TestParseFloat64(t *testing.T) {
	d := ParseFloat64("rate")
	var m map[string]any
	err := d.Decode([]byte("3.14"), &m)
	require.NoError(t, err)
	v, ok := m["rate"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 3.14, v, 0.0001)
}

func TestParseString(t *testing.T) {
	d := ParseString("name")
	var m map[string]any
	err := d.Decode([]byte("hello world"), &m)
	require.NoError(t, err)
	assert.Equal(t, "hello world", m["name"])
}

func TestParseString_Whitespace(t *testing.T) {
	// ParseString trims leading/trailing whitespace before passing to fn
	d := ParseString("name")
	var m map[string]any
	err := d.Decode([]byte("  hello  "), &m)
	require.NoError(t, err)
	assert.Equal(t, "hello", m["name"])
}

func TestParseDuration(t *testing.T) {
	d := ParseDuration("timeout")
	var m map[string]any
	err := d.Decode([]byte("1h2m3s"), &m)
	require.NoError(t, err)
	assert.Equal(t, 1*time.Hour+2*time.Minute+3*time.Second, m["timeout"])
}

func TestParseDuration_Invalid(t *testing.T) {
	d := ParseDuration("timeout")
	var m map[string]any
	err := d.Decode([]byte("notaduration"), &m)
	require.Error(t, err)
}

func TestParseTime(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	d := ParseTime("created_at")
	var m map[string]any
	err := d.Decode([]byte(ts.Format(time.RFC3339)), &m)
	require.NoError(t, err)
	got, ok := m["created_at"].(time.Time)
	require.True(t, ok)
	assert.Equal(t, ts, got)
}

func TestParseTime_Invalid(t *testing.T) {
	d := ParseTime("created_at")
	var m map[string]any
	err := d.Decode([]byte("not-a-time"), &m)
	require.Error(t, err)
}

func TestParseUint(t *testing.T) {
	d := ParseUint("n")
	var m map[string]any
	err := d.Decode([]byte("42"), &m)
	require.NoError(t, err)
	assert.Equal(t, uint(42), m["n"])
}

func TestParseInt64(t *testing.T) {
	d := ParseInt64("bignum")
	var m map[string]any
	err := d.Decode([]byte("1234567890123"), &m)
	require.NoError(t, err)
	assert.Equal(t, int64(1234567890123), m["bignum"])
}

func TestParseInt32(t *testing.T) {
	d := ParseInt32("n")
	var m map[string]any
	err := d.Decode([]byte("100"), &m)
	require.NoError(t, err)
	assert.Equal(t, int32(100), m["n"])
}

func TestParseInt16(t *testing.T) {
	d := ParseInt16("n")
	var m map[string]any
	err := d.Decode([]byte("200"), &m)
	require.NoError(t, err)
	assert.Equal(t, int16(200), m["n"])
}

func TestParseInt8(t *testing.T) {
	d := ParseInt8("n")
	var m map[string]any
	err := d.Decode([]byte("127"), &m)
	require.NoError(t, err)
	assert.Equal(t, int8(127), m["n"])
}

func TestParseUint64(t *testing.T) {
	d := ParseUint64("n")
	var m map[string]any
	err := d.Decode([]byte("18446744073709551615"), &m)
	require.NoError(t, err)
	assert.Equal(t, uint64(18446744073709551615), m["n"])
}

func TestParseUint32(t *testing.T) {
	d := ParseUint32("n")
	var m map[string]any
	err := d.Decode([]byte("4294967295"), &m)
	require.NoError(t, err)
	assert.Equal(t, uint32(4294967295), m["n"])
}

func TestParseUint16(t *testing.T) {
	d := ParseUint16("n")
	var m map[string]any
	err := d.Decode([]byte("65535"), &m)
	require.NoError(t, err)
	assert.Equal(t, uint16(65535), m["n"])
}

func TestParseUint8(t *testing.T) {
	d := ParseUint8("n")
	var m map[string]any
	err := d.Decode([]byte("255"), &m)
	require.NoError(t, err)
	assert.Equal(t, uint8(255), m["n"])
}

func TestParseFloat32(t *testing.T) {
	d := ParseFloat32("n")
	var m map[string]any
	err := d.Decode([]byte("1.5"), &m)
	require.NoError(t, err)
	v, ok := m["n"].(float32)
	require.True(t, ok)
	assert.InDelta(t, 1.5, float64(v), 0.001)
}

func TestParseAs_ResultMapOverwritten(t *testing.T) {
	// Each Decode call produces a fresh single-entry map
	d := ParseInt("x")
	var m map[string]any
	require.NoError(t, d.Decode([]byte("1"), &m))
	assert.Equal(t, 1, m["x"])
	require.NoError(t, d.Decode([]byte("2"), &m))
	assert.Equal(t, 2, m["x"])
	assert.Len(t, m, 1)
}

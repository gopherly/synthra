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

//go:build !integration

package codec

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// EnvVarCodecTestSuite is a test suite for the EnvVar codec.
type EnvVarCodecTestSuite struct {
	suite.Suite
	codec Decoder
}

// SetupTest sets up the test suite.
func (s *EnvVarCodecTestSuite) SetupTest() {
	s.codec = EnvVar
}

// TestEnvVarCodecTestSuite runs the test suite.
func TestEnvVarCodecTestSuite(t *testing.T) {
	suite.Run(t, new(EnvVarCodecTestSuite))
}

// TestDecode_Simple tests the decoding of simple environment variables.
func (s *EnvVarCodecTestSuite) TestDecode_Simple() {
	data := []byte("FOO=bar\nBAZ=qux")
	var v map[string]any
	err := s.codec.Decode(data, &v)
	s.NoError(err)
	s.Equal("bar", v["foo"])
	s.Equal("qux", v["baz"])
}

// TestDecode_Nested tests the decoding of nested environment variables.
func (s *EnvVarCodecTestSuite) TestDecode_Nested() {
	data := []byte("DATABASE_HOST=localhost\nDATABASE_PORT=5432\nDATABASE_USER_NAME=admin")
	var v map[string]any
	err := s.codec.Decode(data, &v)
	s.NoError(err)
	db, ok := v["database"].(map[string]any)
	s.True(ok)
	s.Equal("localhost", db["host"])
	s.Equal("5432", db["port"])
	user, ok := db["user"].(map[string]any)
	s.True(ok)
	s.Equal("admin", user["name"])
}

// TestDecode_Empty tests the decoding of empty environment variables.
func (s *EnvVarCodecTestSuite) TestDecode_Empty() {
	data := []byte("")
	var v map[string]any
	err := s.codec.Decode(data, &v)
	s.NoError(err)
	s.Empty(v)
}

// TestDecode_Malformed tests that a line without '=' returns an error.
func (s *EnvVarCodecTestSuite) TestDecode_Malformed() {
	data := []byte("FOO\nBAR=baz") // FOO has no '='
	var v map[string]any
	err := s.codec.Decode(data, &v)
	s.Error(err)
}

// TestDecode_WrongType tests the decoding of environment variables with
// the wrong type.
func (s *EnvVarCodecTestSuite) TestDecode_WrongType() {
	data := []byte("FOO=bar")
	var v []string // not a *map[string]any
	err := s.codec.Decode(data, &v)
	s.Error(err)
}

// TestDecode_EdgeCases_Whitespace tests the decoding of environment
// variables with whitespace.
func (s *EnvVarCodecTestSuite) TestDecode_EdgeCases_Whitespace() {
	data := []byte("  FOO  =  bar  \n\tBAZ\t=\tqux\t")
	var v map[string]any
	err := s.codec.Decode(data, &v)
	s.NoError(err)
	s.Equal("bar", v["foo"]) // whitespace trimmed from key and value
	s.Equal("qux", v["baz"]) // tabs trimmed
}

// TestDecode_EdgeCases_EmptyKey tests that an empty key returns an error.
func (s *EnvVarCodecTestSuite) TestDecode_EdgeCases_EmptyKey() {
	data := []byte("=value\nFOO=bar")
	var v map[string]any
	err := s.codec.Decode(data, &v)
	s.Error(err)
}

// TestDecode_EdgeCases_UnderscoreKeys tests that double underscores produce
// nested keys (empty parts are filtered out).
func (s *EnvVarCodecTestSuite) TestDecode_EdgeCases_UnderscoreKeys() {
	// FOO__BAR has an empty part between the two underscores; it should
	// be treated the same as FOO_BAR and produce foo.bar.
	data := []byte("FOO__BAR=value4")
	var v map[string]any
	err := s.codec.Decode(data, &v)
	s.NoError(err)

	foo, ok := v["foo"].(map[string]any)
	s.True(ok)
	s.Equal("value4", foo["bar"])
}

// TestDecode_EdgeCases_TypeConflicts tests the decoding of environment
// variables with type conflicts.
func (s *EnvVarCodecTestSuite) TestDecode_EdgeCases_TypeConflicts() {
	// Test type conflicts: scalar vs nested
	data := []byte("FOO=scalar\nFOO_BAR=nested")
	var v map[string]any
	err := s.codec.Decode(data, &v)
	s.NoError(err)

	// The nested structure should overwrite the scalar
	foo, ok := v["foo"].(map[string]any)
	s.True(ok)
	s.Equal("nested", foo["bar"])
}

// TestDecode_EdgeCases_ComplexNesting tests the decoding of environment
// variables with complex nesting.
func (s *EnvVarCodecTestSuite) TestDecode_EdgeCases_ComplexNesting() {
	data := []byte("A_B_C_D=value1\nA_B_E=value2\nA_F=value3")
	var v map[string]any
	err := s.codec.Decode(data, &v)
	s.NoError(err)

	a, ok := v["a"].(map[string]any)
	s.True(ok)

	b, ok := a["b"].(map[string]any)
	s.True(ok)

	c, ok := b["c"].(map[string]any)
	s.True(ok)
	s.Equal("value1", c["d"])
	s.Equal("value2", b["e"])
	s.Equal("value3", a["f"])
}

// TestDecode_EdgeCases_SingleUnderscore tests that a bare underscore key
// produces no entries (all parts are empty after splitting on "_").
func (s *EnvVarCodecTestSuite) TestDecode_EdgeCases_SingleUnderscore() {
	data := []byte("_=value")
	var v map[string]any
	err := s.codec.Decode(data, &v)
	s.NoError(err)
	s.Empty(v)
}

// TestDecode_FailedToCreateNestedMap tests the nested-overwrites-scalar case.
func (s *EnvVarCodecTestSuite) TestDecode_FailedToCreateNestedMap() {
	data := []byte("A=scalar\nA_B=nested")
	var v map[string]any
	err := s.codec.Decode(data, &v)
	if err != nil {
		s.Require().Contains(err.Error(), "failed to create nested map for key:")
		return
	}
	a, ok := v["a"].(map[string]any)
	s.True(ok)
	s.Equal("nested", a["b"])
}

// TestDecode_Comments verifies that comment lines are skipped.
func (s *EnvVarCodecTestSuite) TestDecode_Comments() {
	data := []byte("# this is a comment\nFOO=bar\n# another comment\nBAZ=qux")
	var v map[string]any
	err := s.codec.Decode(data, &v)
	s.NoError(err)
	s.Equal("bar", v["foo"])
	s.Equal("qux", v["baz"])
}

// TestDecode_QuotedValues verifies that quoted values are unquoted.
func (s *EnvVarCodecTestSuite) TestDecode_QuotedValues() {
	data := []byte(`DOUBLE="hello world"` + "\n" + `SINGLE='hello world'`)
	var v map[string]any
	err := s.codec.Decode(data, &v)
	s.NoError(err)
	s.Equal("hello world", v["double"])
	s.Equal("hello world", v["single"])
}

// TestDecode_ExportPrefix verifies that the "export" prefix is stripped.
func (s *EnvVarCodecTestSuite) TestDecode_ExportPrefix() {
	data := []byte("export FOO=bar\nexport BAZ=qux")
	var v map[string]any
	err := s.codec.Decode(data, &v)
	s.NoError(err)
	s.Equal("bar", v["foo"])
	s.Equal("qux", v["baz"])
}

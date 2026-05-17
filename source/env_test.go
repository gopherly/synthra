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

package source

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

// errDecoder is a codec.Decoder that always returns an error.
type errDecoder struct{ err error }

func (d errDecoder) Decode(_ []byte, _ any) error { return d.err }

type OSEnvVarTestSuite struct {
	suite.Suite
}

func (s *OSEnvVarTestSuite) SetupTest() {}

func TestOSEnvVarTestSuite(t *testing.T) {
	suite.Run(t, new(OSEnvVarTestSuite))
}

func (s *OSEnvVarTestSuite) TestLoad_Simple() {
	s.T().Setenv("FOO", "bar")
	s.T().Setenv("BAZ", "qux")

	loader := NewOSEnvVar("")
	conf, err := loader.Load(context.TODO())
	s.NoError(err)
	s.Equal("bar", conf["foo"])
	s.Equal("qux", conf["baz"])
}

func (s *OSEnvVarTestSuite) TestLoad_Nested() {
	s.T().Setenv("DATABASE_HOST", "localhost")
	s.T().Setenv("DATABASE_PORT", "5432")
	s.T().Setenv("DATABASE_USER_NAME", "admin")

	loader := NewOSEnvVar("")
	conf, err := loader.Load(context.TODO())
	s.NoError(err)
	db, ok := conf["database"].(map[string]any)
	s.True(ok)
	s.Equal("localhost", db["host"])
	s.Equal("5432", db["port"])
	user, ok := db["user"].(map[string]any)
	s.True(ok)
	s.Equal("admin", user["name"])
}

func (s *OSEnvVarTestSuite) TestLoad_Empty() {
	// Unset all env vars that might be set by other tests
	os.Clearenv()
	loader := NewOSEnvVar("")
	conf, err := loader.Load(context.TODO())
	s.NoError(err)
	s.Empty(conf)
}

func (s *OSEnvVarTestSuite) TestLoad_Prefix() {
	s.T().Setenv("APP_FOO", "bar")
	s.T().Setenv("APP_BAR", "baz")
	s.T().Setenv("OTHER", "skip")

	loader := NewOSEnvVar("APP_")
	conf, err := loader.Load(context.TODO())
	s.NoError(err)
	s.Equal("bar", conf["foo"])
	s.Equal("baz", conf["bar"])
	s.NotContains(conf, "other")
}

// TestLoad_DecoderError covers the error return path when the decoder
// fails to parse the environment variable data.
func (s *OSEnvVarTestSuite) TestLoad_DecoderError() {
	decodeErr := errors.New("injected decode error")
	loader := &OSEnvVar{
		prefix:  "",
		decoder: errDecoder{err: decodeErr},
	}
	_, err := loader.Load(context.TODO())
	s.Error(err)
	s.ErrorContains(err, "failed to decode environment variables")
	s.ErrorIs(err, decodeErr)
}

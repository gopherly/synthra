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

package synthratest_test

import (
	"context"
	"errors"
	"fmt"

	"gopherly.dev/synthra"
	"gopherly.dev/synthra/synthratest"
)

func ExampleDumper() {
	d := &synthratest.Dumper{}
	ctx := context.Background()
	m := map[string]any{"region": "us-east"}
	if err := d.Dump(ctx, m); err != nil {
		fmt.Println("dump error:", err)
		return
	}
	fmt.Println("calls", d.Calls(), "region", d.Last()["region"])

	// Output: calls 1 region us-east
}

func ExampleErrSource() {
	src := synthratest.ErrSource(errors.New("upstream unavailable"))
	cfg, err := synthra.New(synthra.WithSource(src))
	if err != nil {
		fmt.Println("new:", err)
		return
	}
	err = cfg.Load(context.Background())
	fmt.Println(err != nil)

	// Output: true
}

func ExampleFuncCodec() {
	c := &synthratest.FuncCodec{
		DecodeFunc: func(data []byte, v any) error {
			p, ok := v.(*string)
			if !ok {
				return errors.New("want *string")
			}
			*p = string(data)
			return nil
		},
		EncodeFunc: func(v any) ([]byte, error) {
			s, ok := v.(string)
			if !ok {
				return nil, errors.New("want string")
			}
			return []byte(s), nil
		},
	}
	var out string
	if err := c.Decode([]byte("decoded"), &out); err != nil {
		fmt.Println("decode:", err)
		return
	}
	enc, err := c.Encode("encoded")
	if err != nil {
		fmt.Println("encode:", err)
		return
	}
	fmt.Println(out, string(enc))

	// Output: decoded encoded
}

func ExampleFormat() {
	fmt.Println(synthratest.YAML, synthratest.JSON, synthratest.TOML)

	// Output: yaml json toml
}

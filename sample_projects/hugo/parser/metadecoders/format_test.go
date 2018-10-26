// Copyright 2018 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metadecoders

import (
	"fmt"
	"testing"

	"github.com/gohugoio/hugo/parser/pageparser"

	"github.com/stretchr/testify/require"
)

func TestFormatFromString(t *testing.T) {
	assert := require.New(t)
	for i, test := range []struct {
		s      string
		expect Format
	}{
		{"json", JSON},
		{"yaml", YAML},
		{"yml", YAML},
		{"toml", TOML},
		{"tOMl", TOML},
		{"org", ORG},
		{"foo", ""},
	} {
		assert.Equal(test.expect, FormatFromString(test.s), fmt.Sprintf("t%d", i))
	}
}

func TestFormatFromFrontMatterType(t *testing.T) {
	assert := require.New(t)
	for i, test := range []struct {
		typ    pageparser.ItemType
		expect Format
	}{
		{pageparser.TypeFrontMatterJSON, JSON},
		{pageparser.TypeFrontMatterTOML, TOML},
		{pageparser.TypeFrontMatterYAML, YAML},
		{pageparser.TypeFrontMatterORG, ORG},
		{pageparser.TypeIgnore, ""},
	} {
		assert.Equal(test.expect, FormatFromFrontMatterType(test.typ), fmt.Sprintf("t%d", i))
	}
}

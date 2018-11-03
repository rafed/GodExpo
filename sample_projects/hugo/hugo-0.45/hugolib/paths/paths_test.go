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

package paths

import (
	"testing"

	"github.com/gohugoio/hugo/hugofs"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestNewPaths(t *testing.T) {
	assert := require.New(t)

	v := viper.New()
	fs := hugofs.NewMem(v)

	v.Set("defaultContentLanguageInSubdir", true)
	v.Set("defaultContentLanguage", "no")
	v.Set("multilingual", true)
	v.Set("contentDir", "content")
	v.Set("workingDir", "work")
	v.Set("resourceDir", "resources")
	v.Set("publishDir", "public")

	p, err := New(fs, v)
	assert.NoError(err)

	assert.Equal(true, p.defaultContentLanguageInSubdir)
	assert.Equal("no", p.DefaultContentLanguage)
	assert.Equal(true, p.multilingual)
}

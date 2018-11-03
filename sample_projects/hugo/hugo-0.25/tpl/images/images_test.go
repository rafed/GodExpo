// Copyright 2017 The Hugo Authors. All rights reserved.
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

package images

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
	"testing"

	"github.com/gohugoio/hugo/deps"
	"github.com/gohugoio/hugo/hugofs"
	"github.com/spf13/afero"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tstNoStringer struct{}

var configTests = []struct {
	path   interface{}
	input  []byte
	expect interface{}
}{
	{
		path:  "a.png",
		input: blankImage(10, 10),
		expect: image.Config{
			Width:      10,
			Height:     10,
			ColorModel: color.NRGBAModel,
		},
	},
	{
		path:  "a.png",
		input: blankImage(10, 10),
		expect: image.Config{
			Width:      10,
			Height:     10,
			ColorModel: color.NRGBAModel,
		},
	},
	{
		path:  "b.png",
		input: blankImage(20, 15),
		expect: image.Config{
			Width:      20,
			Height:     15,
			ColorModel: color.NRGBAModel,
		},
	},
	{
		path:  "a.png",
		input: blankImage(20, 15),
		expect: image.Config{
			Width:      10,
			Height:     10,
			ColorModel: color.NRGBAModel,
		},
	},
	// errors
	{path: tstNoStringer{}, expect: false},
	{path: "non-existent.png", expect: false},
	{path: "", expect: false},
}

func TestNSConfig(t *testing.T) {
	t.Parallel()

	v := viper.New()
	v.Set("workingDir", "/a/b")

	ns := New(&deps.Deps{Fs: hugofs.NewMem(v)})

	for i, test := range configTests {
		errMsg := fmt.Sprintf("[%d] %s", i, test.path)

		// check for expected errors early to avoid writing files
		if b, ok := test.expect.(bool); ok && !b {
			_, err := ns.Config(interface{}(test.path))
			require.Error(t, err, errMsg)
			continue
		}

		// cast path to string for afero.WriteFile
		sp, err := cast.ToStringE(test.path)
		require.NoError(t, err, errMsg)
		afero.WriteFile(ns.deps.Fs.Source, filepath.Join(v.GetString("workingDir"), sp), test.input, 0755)

		result, err := ns.Config(test.path)

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
		assert.NotEqual(t, 0, len(ns.cache), errMsg)
	}
}

func blankImage(width, height int) []byte {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

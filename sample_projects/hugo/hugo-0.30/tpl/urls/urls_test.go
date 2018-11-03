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

package urls

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/gohugoio/hugo/deps"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ns = New(&deps.Deps{Cfg: viper.New()})

type tstNoStringer struct{}

func TestParse(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		rawurl interface{}
		expect interface{}
	}{
		{
			"http://www.google.com",
			&url.URL{
				Scheme: "http",
				Host:   "www.google.com",
			},
		},
		{
			"http://j@ne:password@google.com",
			&url.URL{
				Scheme: "http",
				User:   url.UserPassword("j@ne", "password"),
				Host:   "google.com",
			},
		},
		// errors
		{tstNoStringer{}, false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.Parse(test.rawurl)

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

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

package strings

import (
	"fmt"
	"html/template"
	"testing"

	"github.com/gohugoio/hugo/deps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ns = New(&deps.Deps{})

type tstNoStringer struct{}

func TestChomp(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		expect interface{}
	}{
		{"\n a\n", template.HTML("\n a")},
		{"\n a\n\n", template.HTML("\n a")},
		{"\n a\r\n", template.HTML("\n a")},
		{"\n a\n\r\n", template.HTML("\n a")},
		{"\n a\r\r", template.HTML("\n a")},
		{"\n a\r", template.HTML("\n a")},
		// errors
		{tstNoStringer{}, false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.Chomp(test.s)

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestContains(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		substr interface{}
		expect bool
		isErr  bool
	}{
		{"", "", true, false},
		{"123", "23", true, false},
		{"123", "234", false, false},
		{"123", "", true, false},
		{"", "a", false, false},
		{123, "23", true, false},
		{123, "234", false, false},
		{123, "", true, false},
		{template.HTML("123"), []byte("23"), true, false},
		{template.HTML("123"), []byte("234"), false, false},
		{template.HTML("123"), []byte(""), true, false},
		// errors
		{"", tstNoStringer{}, false, true},
		{tstNoStringer{}, "", false, true},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.Contains(test.s, test.substr)

		if test.isErr {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestContainsAny(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		substr interface{}
		expect bool
		isErr  bool
	}{
		{"", "", false, false},
		{"", "1", false, false},
		{"", "123", false, false},
		{"1", "", false, false},
		{"1", "1", true, false},
		{"111", "1", true, false},
		{"123", "789", false, false},
		{"123", "729", true, false},
		{"a☺b☻c☹d", "uvw☻xyz", true, false},
		{1, "", false, false},
		{1, "1", true, false},
		{111, "1", true, false},
		{123, "789", false, false},
		{123, "729", true, false},
		{[]byte("123"), template.HTML("789"), false, false},
		{[]byte("123"), template.HTML("729"), true, false},
		{[]byte("a☺b☻c☹d"), template.HTML("uvw☻xyz"), true, false},
		// errors
		{"", tstNoStringer{}, false, true},
		{tstNoStringer{}, "", false, true},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.ContainsAny(test.s, test.substr)

		if test.isErr {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestCountRunes(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		expect interface{}
	}{
		{"foo bar", 6},
		{"旁边", 2},
		{`<div class="test">旁边</div>`, 2},
		// errors
		{tstNoStringer{}, false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test.s)

		result, err := ns.CountRunes(test.s)

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestCountWords(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		expect interface{}
	}{
		{"Do Be Do Be Do", 5},
		{"旁边", 2},
		{`<div class="test">旁边</div>`, 2},
		// errors
		{tstNoStringer{}, false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test.s)

		result, err := ns.CountWords(test.s)

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestHasPrefix(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		prefix interface{}
		expect interface{}
		isErr  bool
	}{
		{"abcd", "ab", true, false},
		{"abcd", "cd", false, false},
		{template.HTML("abcd"), "ab", true, false},
		{template.HTML("abcd"), "cd", false, false},
		{template.HTML("1234"), 12, true, false},
		{template.HTML("1234"), 34, false, false},
		{[]byte("abcd"), "ab", true, false},
		// errors
		{"", tstNoStringer{}, false, true},
		{tstNoStringer{}, "", false, true},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.HasPrefix(test.s, test.prefix)

		if test.isErr {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestHasSuffix(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		suffix interface{}
		expect interface{}
		isErr  bool
	}{
		{"abcd", "cd", true, false},
		{"abcd", "ab", false, false},
		{template.HTML("abcd"), "cd", true, false},
		{template.HTML("abcd"), "ab", false, false},
		{template.HTML("1234"), 34, true, false},
		{template.HTML("1234"), 12, false, false},
		{[]byte("abcd"), "cd", true, false},
		// errors
		{"", tstNoStringer{}, false, true},
		{tstNoStringer{}, "", false, true},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.HasSuffix(test.s, test.suffix)

		if test.isErr {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestReplace(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		old    interface{}
		new    interface{}
		expect interface{}
	}{
		{"aab", "a", "b", "bbb"},
		{"11a11", 1, 2, "22a22"},
		{12345, 1, 2, "22345"},
		// errors
		{tstNoStringer{}, "a", "b", false},
		{"a", tstNoStringer{}, "b", false},
		{"a", "b", tstNoStringer{}, false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.Replace(test.s, test.old, test.new)

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestSliceString(t *testing.T) {
	t.Parallel()

	var err error
	for i, test := range []struct {
		v1     interface{}
		v2     interface{}
		v3     interface{}
		expect interface{}
	}{
		{"abc", 1, 2, "b"},
		{"abc", 1, 3, "bc"},
		{"abcdef", 1, int8(3), "bc"},
		{"abcdef", 1, int16(3), "bc"},
		{"abcdef", 1, int32(3), "bc"},
		{"abcdef", 1, int64(3), "bc"},
		{"abc", 0, 1, "a"},
		{"abcdef", nil, nil, "abcdef"},
		{"abcdef", 0, 6, "abcdef"},
		{"abcdef", 0, 2, "ab"},
		{"abcdef", 2, nil, "cdef"},
		{"abcdef", int8(2), nil, "cdef"},
		{"abcdef", int16(2), nil, "cdef"},
		{"abcdef", int32(2), nil, "cdef"},
		{"abcdef", int64(2), nil, "cdef"},
		{123, 1, 3, "23"},
		{"abcdef", 6, nil, false},
		{"abcdef", 4, 7, false},
		{"abcdef", -1, nil, false},
		{"abcdef", -1, 7, false},
		{"abcdef", 1, -1, false},
		{tstNoStringer{}, 0, 1, false},
		{"ĀĀĀ", 0, 1, "Ā"}, // issue #1333
		{"a", t, nil, false},
		{"a", 1, t, false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		var result string
		if test.v2 == nil {
			result, err = ns.SliceString(test.v1)
		} else if test.v3 == nil {
			result, err = ns.SliceString(test.v1, test.v2)
		} else {
			result, err = ns.SliceString(test.v1, test.v2, test.v3)
		}

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}

	// Too many arguments
	_, err = ns.SliceString("a", 1, 2, 3)
	if err == nil {
		t.Errorf("Should have errored")
	}
}

func TestSplit(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		v1     interface{}
		v2     string
		expect interface{}
	}{
		{"a, b", ", ", []string{"a", "b"}},
		{"a & b & c", " & ", []string{"a", "b", "c"}},
		{"http://example.com", "http://", []string{"", "example.com"}},
		{123, "2", []string{"1", "3"}},
		{tstNoStringer{}, ",", false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.Split(test.v1, test.v2)

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestSubstr(t *testing.T) {
	t.Parallel()

	var err error
	var n int
	for i, test := range []struct {
		v1     interface{}
		v2     interface{}
		v3     interface{}
		expect interface{}
	}{
		{"abc", 1, 2, "bc"},
		{"abc", 0, 1, "a"},
		{"abcdef", -1, 2, "ef"},
		{"abcdef", -3, 3, "bcd"},
		{"abcdef", 0, -1, "abcde"},
		{"abcdef", 2, -1, "cde"},
		{"abcdef", 4, -4, false},
		{"abcdef", 7, 1, false},
		{"abcdef", 1, 100, "bcdef"},
		{"abcdef", -100, 3, "abc"},
		{"abcdef", -3, -1, "de"},
		{"abcdef", 2, nil, "cdef"},
		{"abcdef", int8(2), nil, "cdef"},
		{"abcdef", int16(2), nil, "cdef"},
		{"abcdef", int32(2), nil, "cdef"},
		{"abcdef", int64(2), nil, "cdef"},
		{"abcdef", 2, int8(3), "cde"},
		{"abcdef", 2, int16(3), "cde"},
		{"abcdef", 2, int32(3), "cde"},
		{"abcdef", 2, int64(3), "cde"},
		{123, 1, 3, "23"},
		{1.2e3, 0, 4, "1200"},
		{tstNoStringer{}, 0, 1, false},
		{"abcdef", 2.0, nil, "cdef"},
		{"abcdef", 2.0, 2, "cd"},
		{"abcdef", 2, 2.0, "cd"},
		{"ĀĀĀ", 1, 2, "ĀĀ"}, // # issue 1333
		{"abcdef", "doo", nil, false},
		{"abcdef", "doo", "doo", false},
		{"abcdef", 1, "doo", false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		var result string
		n = i

		if test.v3 == nil {
			result, err = ns.Substr(test.v1, test.v2)
		} else {
			result, err = ns.Substr(test.v1, test.v2, test.v3)
		}

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}

	n++
	_, err = ns.Substr("abcdef")
	if err == nil {
		t.Errorf("[%d] Substr didn't return an expected error", n)
	}

	n++
	_, err = ns.Substr("abcdef", 1, 2, 3)
	if err == nil {
		t.Errorf("[%d] Substr didn't return an expected error", n)
	}
}

func TestTitle(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		expect interface{}
	}{
		{"test", "Test"},
		{template.HTML("hypertext"), "Hypertext"},
		{[]byte("bytes"), "Bytes"},
		// errors
		{tstNoStringer{}, false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.Title(test.s)

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestToLower(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		expect interface{}
	}{
		{"TEST", "test"},
		{template.HTML("LoWeR"), "lower"},
		{[]byte("BYTES"), "bytes"},
		// errors
		{tstNoStringer{}, false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.ToLower(test.s)

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestToUpper(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		expect interface{}
	}{
		{"test", "TEST"},
		{template.HTML("UpPeR"), "UPPER"},
		{[]byte("bytes"), "BYTES"},
		// errors
		{tstNoStringer{}, false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.ToUpper(test.s)

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestTrim(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		cutset interface{}
		expect interface{}
	}{
		{"abba", "a", "bb"},
		{"abba", "ab", ""},
		{"<tag>", "<>", "tag"},
		{`"quote"`, `"`, "quote"},
		{1221, "1", "22"},
		{1221, "12", ""},
		{template.HTML("<tag>"), "<>", "tag"},
		{[]byte("<tag>"), "<>", "tag"},
		// errors
		{"", tstNoStringer{}, false},
		{tstNoStringer{}, "", false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.Trim(test.s, test.cutset)

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestTrimPrefix(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		prefix interface{}
		expect interface{}
	}{
		{"aabbaa", "a", "abbaa"},
		{"aabb", "b", "aabb"},
		{1234, "12", "34"},
		{1234, "34", "1234"},
		// errors
		{"", tstNoStringer{}, false},
		{tstNoStringer{}, "", false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.TrimPrefix(test.s, test.prefix)

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

func TestTrimSuffix(t *testing.T) {
	t.Parallel()

	for i, test := range []struct {
		s      interface{}
		suffix interface{}
		expect interface{}
	}{
		{"aabbaa", "a", "aabba"},
		{"aabb", "b", "aab"},
		{1234, "12", "1234"},
		{1234, "34", "12"},
		// errors
		{"", tstNoStringer{}, false},
		{tstNoStringer{}, "", false},
	} {
		errMsg := fmt.Sprintf("[%d] %v", i, test)

		result, err := ns.TrimSuffix(test.s, test.suffix)

		if b, ok := test.expect.(bool); ok && !b {
			require.Error(t, err, errMsg)
			continue
		}

		require.NoError(t, err, errMsg)
		assert.Equal(t, test.expect, result, errMsg)
	}
}

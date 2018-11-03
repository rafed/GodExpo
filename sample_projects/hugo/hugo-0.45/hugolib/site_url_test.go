// Copyright 2016 The Hugo Authors. All rights reserved.
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

package hugolib

import (
	"fmt"
	"path/filepath"
	"testing"

	"html/template"

	"github.com/gohugoio/hugo/deps"
	"github.com/stretchr/testify/require"
)

const slugDoc1 = "---\ntitle: slug doc 1\nslug: slug-doc-1\naliases:\n - sd1/foo/\n - sd2\n - sd3/\n - sd4.html\n---\nslug doc 1 content\n"

const slugDoc2 = `---
title: slug doc 2
slug: slug-doc-2
---
slug doc 2 content
`

var urlFakeSource = [][2]string{
	{filepath.FromSlash("content/blue/doc1.md"), slugDoc1},
	{filepath.FromSlash("content/blue/doc2.md"), slugDoc2},
}

// Issue #1105
func TestShouldNotAddTrailingSlashToBaseURL(t *testing.T) {
	t.Parallel()
	for i, this := range []struct {
		in       string
		expected string
	}{
		{"http://base.com/", "http://base.com/"},
		{"http://base.com/sub/", "http://base.com/sub/"},
		{"http://base.com/sub", "http://base.com/sub"},
		{"http://base.com", "http://base.com"}} {

		cfg, fs := newTestCfg()
		cfg.Set("baseURL", this.in)
		d := deps.DepsCfg{Cfg: cfg, Fs: fs}
		s, err := NewSiteForCfg(d)
		require.NoError(t, err)
		require.NoError(t, s.initializeSiteInfo())

		if s.Info.BaseURL() != template.URL(this.expected) {
			t.Errorf("[%d] got %s expected %s", i, s.Info.BaseURL(), this.expected)
		}
	}
}

func TestPageCount(t *testing.T) {
	t.Parallel()
	cfg, fs := newTestCfg()
	cfg.Set("uglyURLs", false)
	cfg.Set("paginate", 10)

	writeSourcesToSource(t, "", fs, urlFakeSource...)
	s := buildSingleSite(t, deps.DepsCfg{Fs: fs, Cfg: cfg}, BuildCfg{})

	_, err := s.Fs.Destination.Open("public/blue")
	if err != nil {
		t.Errorf("No indexed rendered.")
	}

	for _, pth := range []string{
		"public/sd1/foo/index.html",
		"public/sd2/index.html",
		"public/sd3/index.html",
		"public/sd4.html",
	} {
		if _, err := s.Fs.Destination.Open(filepath.FromSlash(pth)); err != nil {
			t.Errorf("No alias rendered: %s", pth)
		}
	}
}

func TestUglyURLsPerSection(t *testing.T) {
	t.Parallel()

	assert := require.New(t)

	const dt = `---
title: Do not go gentle into that good night
---

Wild men who caught and sang the sun in flight,
And learn, too late, they grieved it on its way,
Do not go gentle into that good night.

`

	cfg, fs := newTestCfg()

	cfg.Set("uglyURLs", map[string]bool{
		"sect2": true,
	})

	writeSource(t, fs, filepath.Join("content", "sect1", "p1.md"), dt)
	writeSource(t, fs, filepath.Join("content", "sect2", "p2.md"), dt)

	s := buildSingleSite(t, deps.DepsCfg{Fs: fs, Cfg: cfg}, BuildCfg{SkipRender: true})

	assert.Len(s.RegularPages, 2)

	notUgly := s.getPage(KindPage, "sect1/p1.md")
	assert.NotNil(notUgly)
	assert.Equal("sect1", notUgly.Section())
	assert.Equal("/sect1/p1/", notUgly.RelPermalink())

	ugly := s.getPage(KindPage, "sect2/p2.md")
	assert.NotNil(ugly)
	assert.Equal("sect2", ugly.Section())
	assert.Equal("/sect2/p2.html", ugly.RelPermalink())
}

func TestSectionWithURLInFrontMatter(t *testing.T) {
	t.Parallel()

	assert := require.New(t)

	const st = `---
title: Do not go gentle into that good night
url: %s
---

Wild men who caught and sang the sun in flight,
And learn, too late, they grieved it on its way,
Do not go gentle into that good night.

`

	const pt = `---
title: Wild men who caught and sang the sun in flight
---

Wild men who caught and sang the sun in flight,
And learn, too late, they grieved it on its way,
Do not go gentle into that good night.

`

	cfg, fs := newTestCfg()
	th := testHelper{cfg, fs, t}

	cfg.Set("paginate", 1)

	writeSource(t, fs, filepath.Join("content", "sect1", "_index.md"), fmt.Sprintf(st, "/ss1/"))
	writeSource(t, fs, filepath.Join("content", "sect2", "_index.md"), fmt.Sprintf(st, "/ss2/"))

	for i := 0; i < 5; i++ {
		writeSource(t, fs, filepath.Join("content", "sect1", fmt.Sprintf("p%d.md", i+1)), pt)
		writeSource(t, fs, filepath.Join("content", "sect2", fmt.Sprintf("p%d.md", i+1)), pt)
	}

	writeSource(t, fs, filepath.Join("layouts", "_default", "single.html"), "<html><body>{{.Content}}</body></html>")
	writeSource(t, fs, filepath.Join("layouts", "_default", "list.html"),
		"<html><body>P{{.Paginator.PageNumber}}|URL: {{.Paginator.URL}}|{{ if .Paginator.HasNext }}Next: {{.Paginator.Next.URL }}{{ end }}</body></html>")

	s := buildSingleSite(t, deps.DepsCfg{Fs: fs, Cfg: cfg}, BuildCfg{})

	assert.Len(s.RegularPages, 10)

	sect1 := s.getPage(KindSection, "sect1")
	assert.NotNil(sect1)
	assert.Equal("/ss1/", sect1.RelPermalink())
	th.assertFileContent(filepath.Join("public", "ss1", "index.html"), "P1|URL: /ss1/|Next: /ss1/page/2/")
	th.assertFileContent(filepath.Join("public", "ss1", "page", "2", "index.html"), "P2|URL: /ss1/page/2/|Next: /ss1/page/3/")

}

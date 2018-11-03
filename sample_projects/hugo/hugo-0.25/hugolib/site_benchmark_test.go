// Copyright 2017-present The Hugo Authors. All rights reserved.
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
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

type siteBuildingBenchmarkConfig struct {
	Frontmatter  string
	NumPages     int
	RootSections int
	Render       bool
	Shortcodes   bool
	NumTags      int
	TagsPerPage  int
}

func (s siteBuildingBenchmarkConfig) String() string {
	// Make it comma separated with no spaces, so it is both Bash and regexp friendly.
	// To make it a short as possible, we only shows bools when enabled and ints when >= 0 (RootSections > 1)
	sep := ","
	id := s.Frontmatter + sep
	if s.RootSections > 1 {
		id += fmt.Sprintf("num_root_sections=%d%s", s.RootSections, sep)
	}
	id += fmt.Sprintf("num_pages=%d%s", s.NumPages, sep)

	if s.NumTags > 0 {
		id += fmt.Sprintf("num_tags=%d%s", s.NumTags, sep)
	}

	if s.TagsPerPage > 0 {
		id += fmt.Sprintf("tags_per_page=%d%s", s.TagsPerPage, sep)
	}

	if s.Shortcodes {
		id += "shortcodes" + sep
	}

	if s.Render {
		id += "render" + sep
	}

	return strings.TrimSuffix(id, sep)

}

func BenchmarkSiteBuilding(b *testing.B) {
	var conf siteBuildingBenchmarkConfig
	for _, frontmatter := range []string{"YAML", "TOML"} {
		conf.Frontmatter = frontmatter
		for _, rootSections := range []int{1, 5} {
			conf.RootSections = rootSections
			for _, numTags := range []int{0, 1, 10, 20, 50, 100, 500, 1000, 5000} {
				conf.NumTags = numTags
				for _, tagsPerPage := range []int{0, 1, 5, 20, 50, 80} {
					conf.TagsPerPage = tagsPerPage
					for _, numPages := range []int{1, 10, 100, 500, 1000, 5000, 10000} {
						conf.NumPages = numPages
						for _, render := range []bool{false, true} {
							conf.Render = render
							for _, shortcodes := range []bool{false, true} {
								conf.Shortcodes = shortcodes
								doBenchMarkSiteBuilding(conf, b)
							}
						}
					}
				}
			}
		}
	}
}

func doBenchMarkSiteBuilding(conf siteBuildingBenchmarkConfig, b *testing.B) {
	b.Run(conf.String(), func(b *testing.B) {
		sites := createHugoBenchmarkSites(b, b.N, conf)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			h := sites[0]

			err := h.Build(BuildCfg{SkipRender: !conf.Render})
			if err != nil {
				b.Fatal(err)
			}

			// Try to help the GC
			sites[0] = nil
			sites = sites[1:len(sites)]
		}
	})
}

func createHugoBenchmarkSites(b *testing.B, count int, cfg siteBuildingBenchmarkConfig) []*HugoSites {
	someMarkdown := `
An h1 header
============

Paragraphs are separated by a blank line.

2nd paragraph. *Italic* and **bold**. Itemized lists
look like:

  * this one
  * that one
  * the other one

Note that --- not considering the asterisk --- the actual text
content starts at 4-columns in.

> Block quotes are
> written like so.
>
> They can span multiple paragraphs,
> if you like.

Use 3 dashes for an em-dash. Use 2 dashes for ranges (ex., "it's all
in chapters 12--14"). Three dots ... will be converted to an ellipsis.
Unicode is supported. ☺
`

	someMarkdownWithShortCode := someMarkdown + `

{{< myShortcode >}}

`

	pageTemplateTOML := `+++
title = "%s"
tags = %s
+++
%s

`

	pageTemplateYAML := `---
title: "%s"
tags:
%s
---
%s

`

	siteConfig := `
baseURL = "http://example.com/blog"

paginate = 10
defaultContentLanguage = "en"

[Taxonomies]
tag = "tags"
category = "categories"
`

	numTags := cfg.NumTags

	if cfg.TagsPerPage > numTags {
		numTags = cfg.TagsPerPage
	}

	var (
		contentPagesContent [3]string
		tags                = make([]string, numTags)
		pageTemplate        string
	)

	for i := 0; i < numTags; i++ {
		tags[i] = fmt.Sprintf("Hugo %d", i+1)
	}

	var tagsStr string

	if cfg.Shortcodes {
		contentPagesContent = [3]string{
			someMarkdownWithShortCode,
			strings.Repeat(someMarkdownWithShortCode, 2),
			strings.Repeat(someMarkdownWithShortCode, 3),
		}
	} else {
		contentPagesContent = [3]string{
			someMarkdown,
			strings.Repeat(someMarkdown, 2),
			strings.Repeat(someMarkdown, 3),
		}
	}

	sites := make([]*HugoSites, count)
	for i := 0; i < count; i++ {
		// Maybe consider reusing the Source fs
		mf := afero.NewMemMapFs()
		th, h := newTestSitesFromConfig(b, mf, siteConfig,
			"layouts/_default/single.html", `Single HTML|{{ .Title }}|{{ .Content }}`,
			"layouts/_default/list.html", `List HTML|{{ .Title }}|{{ .Content }}`,
			"layouts/shortcodes/myShortcode.html", `<p>MyShortcode</p>`)

		fs := th.Fs

		pagesPerSection := cfg.NumPages / cfg.RootSections

		for i := 0; i < cfg.RootSections; i++ {
			for j := 0; j < pagesPerSection; j++ {
				var tagsSlice []string

				if numTags > 0 {
					tagsStart := rand.Intn(numTags) - cfg.TagsPerPage
					if tagsStart < 0 {
						tagsStart = 0
					}
					tagsSlice = tags[tagsStart : tagsStart+cfg.TagsPerPage]
				}

				if cfg.Frontmatter == "TOML" {
					pageTemplate = pageTemplateTOML
					tagsStr = "[]"
					if cfg.TagsPerPage > 0 {
						tagsStr = strings.Replace(fmt.Sprintf("%q", tagsSlice), " ", ", ", -1)
					}
				} else {
					// YAML
					pageTemplate = pageTemplateYAML
					for _, tag := range tagsSlice {
						tagsStr += "\n- " + tag
					}
				}

				content := fmt.Sprintf(pageTemplate, fmt.Sprintf("Title%d_%d", i, j), tagsStr, contentPagesContent[rand.Intn(3)])

				writeSource(b, fs, filepath.Join("content", fmt.Sprintf("sect%d", i), fmt.Sprintf("page%d.md", j)), content)
			}
		}

		sites[i] = h
	}

	return sites
}

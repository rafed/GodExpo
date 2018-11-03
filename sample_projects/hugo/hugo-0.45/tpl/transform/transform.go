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

package transform

import (
	"bytes"
	"html"
	"html/template"

	"github.com/gohugoio/hugo/deps"
	"github.com/gohugoio/hugo/helpers"
	"github.com/spf13/cast"
)

// New returns a new instance of the transform-namespaced template functions.
func New(deps *deps.Deps) *Namespace {
	return &Namespace{
		deps: deps,
	}
}

// Namespace provides template functions for the "transform" namespace.
type Namespace struct {
	deps *deps.Deps
}

// Emojify returns a copy of s with all emoji codes replaced with actual emojis.
//
// See http://www.emoji-cheat-sheet.com/
func (ns *Namespace) Emojify(s interface{}) (template.HTML, error) {
	ss, err := cast.ToStringE(s)
	if err != nil {
		return "", err
	}

	return template.HTML(helpers.Emojify([]byte(ss))), nil
}

// Highlight returns a copy of s as an HTML string with syntax
// highlighting applied.
func (ns *Namespace) Highlight(s interface{}, lang, opts string) (template.HTML, error) {
	ss, err := cast.ToStringE(s)
	if err != nil {
		return "", err
	}

	highlighted, _ := ns.deps.ContentSpec.Highlight(ss, lang, opts)
	return template.HTML(highlighted), nil
}

// HTMLEscape returns a copy of s with reserved HTML characters escaped.
func (ns *Namespace) HTMLEscape(s interface{}) (string, error) {
	ss, err := cast.ToStringE(s)
	if err != nil {
		return "", err
	}

	return html.EscapeString(ss), nil
}

// HTMLUnescape returns a copy of with HTML escape requences converted to plain
// text.
func (ns *Namespace) HTMLUnescape(s interface{}) (string, error) {
	ss, err := cast.ToStringE(s)
	if err != nil {
		return "", err
	}

	return html.UnescapeString(ss), nil
}

var (
	markdownTrimPrefix         = []byte("<p>")
	markdownTrimSuffix         = []byte("</p>\n")
	markdownParagraphIndicator = []byte("<p")
)

// Markdownify renders a given input from Markdown to HTML.
func (ns *Namespace) Markdownify(s interface{}) (template.HTML, error) {
	ss, err := cast.ToStringE(s)
	if err != nil {
		return "", err
	}

	m := ns.deps.ContentSpec.RenderBytes(
		&helpers.RenderingContext{
			Cfg:     ns.deps.Cfg,
			Content: []byte(ss),
			PageFmt: "markdown",
			Config:  ns.deps.ContentSpec.BlackFriday,
		},
	)

	// Strip if this is a short inline type of text.
	first := bytes.Index(m, markdownParagraphIndicator)
	last := bytes.LastIndex(m, markdownParagraphIndicator)
	if first == last {
		m = bytes.TrimPrefix(m, markdownTrimPrefix)
		m = bytes.TrimSuffix(m, markdownTrimSuffix)
	}

	return template.HTML(m), nil
}

// Plainify returns a copy of s with all HTML tags removed.
func (ns *Namespace) Plainify(s interface{}) (string, error) {
	ss, err := cast.ToStringE(s)
	if err != nil {
		return "", err
	}

	return helpers.StripHTML(ss), nil
}

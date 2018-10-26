// Copyright 2015 The Hugo Authors. All rights reserved.
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
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gohugoio/hugo/helpers"
)

// pathPattern represents a string which builds up a URL from attributes
type pathPattern string

// pageToPermaAttribute is the type of a function which, given a page and a tag
// can return a string to go in that position in the page (or an error)
type pageToPermaAttribute func(*Page, string) (string, error)

// PermalinkOverrides maps a section name to a PathPattern
type PermalinkOverrides map[string]pathPattern

// knownPermalinkAttributes maps :tags in a permalink specification to a
// function which, given a page and the tag, returns the resulting string
// to be used to replace that tag.
var knownPermalinkAttributes map[string]pageToPermaAttribute

var attributeRegexp *regexp.Regexp

// validate determines if a PathPattern is well-formed
func (pp pathPattern) validate() bool {
	fragments := strings.Split(string(pp[1:]), "/")
	var bail = false
	for i := range fragments {
		if bail {
			return false
		}
		if len(fragments[i]) == 0 {
			bail = true
			continue
		}

		matches := attributeRegexp.FindAllStringSubmatch(fragments[i], -1)
		if matches == nil {
			continue
		}

		for _, match := range matches {
			k := strings.ToLower(match[0][1:])
			if _, ok := knownPermalinkAttributes[k]; !ok {
				return false
			}
		}
	}
	return true
}

type permalinkExpandError struct {
	pattern pathPattern
	section string
	err     error
}

func (pee *permalinkExpandError) Error() string {
	return fmt.Sprintf("error expanding %q section %q: %s", string(pee.pattern), pee.section, pee.err)
}

var (
	errPermalinkIllFormed        = errors.New("permalink ill-formed")
	errPermalinkAttributeUnknown = errors.New("permalink attribute not recognised")
)

// Expand on a PathPattern takes a Page and returns the fully expanded Permalink
// or an error explaining the failure.
func (pp pathPattern) Expand(p *Page) (string, error) {
	if !pp.validate() {
		return "", &permalinkExpandError{pattern: pp, section: "<all>", err: errPermalinkIllFormed}
	}
	sections := strings.Split(string(pp), "/")
	for i, field := range sections {
		if len(field) == 0 {
			continue
		}

		matches := attributeRegexp.FindAllStringSubmatch(field, -1)

		if matches == nil {
			continue
		}

		newField := field

		for _, match := range matches {
			attr := match[0][1:]
			callback, ok := knownPermalinkAttributes[attr]

			if !ok {
				return "", &permalinkExpandError{pattern: pp, section: strconv.Itoa(i), err: errPermalinkAttributeUnknown}
			}

			newAttr, err := callback(p, attr)

			if err != nil {
				return "", &permalinkExpandError{pattern: pp, section: strconv.Itoa(i), err: err}
			}

			newField = strings.Replace(newField, match[0], newAttr, 1)
		}

		sections[i] = newField
	}
	return strings.Join(sections, "/"), nil
}

func pageToPermalinkDate(p *Page, dateField string) (string, error) {
	// a Page contains a Node which provides a field Date, time.Time
	switch dateField {
	case "year":
		return strconv.Itoa(p.Date.Year()), nil
	case "month":
		return fmt.Sprintf("%02d", int(p.Date.Month())), nil
	case "monthname":
		return p.Date.Month().String(), nil
	case "day":
		return fmt.Sprintf("%02d", p.Date.Day()), nil
	case "weekday":
		return strconv.Itoa(int(p.Date.Weekday())), nil
	case "weekdayname":
		return p.Date.Weekday().String(), nil
	case "yearday":
		return strconv.Itoa(p.Date.YearDay()), nil
	}
	//TODO: support classic strftime escapes too
	// (and pass those through despite not being in the map)
	panic("coding error: should not be here")
}

// pageToPermalinkTitle returns the URL-safe form of the title
func pageToPermalinkTitle(p *Page, _ string) (string, error) {
	if p.Kind == KindTaxonomy {
		// Taxonomies are allowed to have '/' characters, so don't normalize
		// them with MakeSegment.
		return p.s.PathSpec.MakePathSanitized(p.title), nil
	}

	return p.s.PathSpec.MakeSegment(p.title), nil
}

// pageToPermalinkFilename returns the URL-safe form of the filename
func pageToPermalinkFilename(p *Page, _ string) (string, error) {
	name := p.File.TranslationBaseName()
	if name == "index" {
		// Page bundles; the directory name will hopefully have a better name.
		dir := strings.TrimSuffix(p.File.Dir(), helpers.FilePathSeparator)
		_, name = filepath.Split(dir)
	}

	return p.s.PathSpec.MakeSegment(name), nil
}

// if the page has a slug, return the slug, else return the title
func pageToPermalinkSlugElseTitle(p *Page, a string) (string, error) {
	if p.Slug != "" {
		// Don't start or end with a -
		// TODO(bep) this doesn't look good... Set the Slug once.
		if strings.HasPrefix(p.Slug, "-") {
			p.Slug = p.Slug[1:len(p.Slug)]
		}

		if strings.HasSuffix(p.Slug, "-") {
			p.Slug = p.Slug[0 : len(p.Slug)-1]
		}
		return p.s.PathSpec.MakeSegment(p.Slug), nil
	}
	return pageToPermalinkTitle(p, a)
}

func pageToPermalinkSection(p *Page, _ string) (string, error) {
	// Page contains Node contains URLPath which has Section
	return p.s.PathSpec.MakeSegment(p.Section()), nil
}

func pageToPermalinkSections(p *Page, _ string) (string, error) {
	// TODO(bep) we have some superflous URLize in this file, but let's
	// deal with that later.

	cs := p.CurrentSection()
	if cs == nil {
		return "", errors.New("\":sections\" attribute requires parent page but is nil")
	}

	sections := make([]string, len(cs.sections))
	for i := range cs.sections {
		sections[i] = p.s.PathSpec.MakeSegment(cs.sections[i])
	}
	return path.Join(sections...), nil
}

func init() {
	knownPermalinkAttributes = map[string]pageToPermaAttribute{
		"year":        pageToPermalinkDate,
		"month":       pageToPermalinkDate,
		"monthname":   pageToPermalinkDate,
		"day":         pageToPermalinkDate,
		"weekday":     pageToPermalinkDate,
		"weekdayname": pageToPermalinkDate,
		"yearday":     pageToPermalinkDate,
		"section":     pageToPermalinkSection,
		"sections":    pageToPermalinkSections,
		"title":       pageToPermalinkTitle,
		"slug":        pageToPermalinkSlugElseTitle,
		"filename":    pageToPermalinkFilename,
	}

	attributeRegexp = regexp.MustCompile(`:\w+`)
}

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
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"unicode"

	"github.com/gohugoio/hugo/related"

	"github.com/bep/gitmap"

	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/resource"

	"github.com/gohugoio/hugo/output"
	"github.com/gohugoio/hugo/parser"
	"github.com/mitchellh/mapstructure"

	"html/template"
	"io"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	bp "github.com/gohugoio/hugo/bufferpool"
	"github.com/gohugoio/hugo/compare"
	"github.com/gohugoio/hugo/source"
	"github.com/spf13/cast"
)

var (
	cjk = regexp.MustCompile(`\p{Han}|\p{Hangul}|\p{Hiragana}|\p{Katakana}`)

	// This is all the kinds we can expect to find in .Site.Pages.
	allKindsInPages = []string{KindPage, KindHome, KindSection, KindTaxonomy, KindTaxonomyTerm}

	allKinds = append(allKindsInPages, []string{kindRSS, kindSitemap, kindRobotsTXT, kind404}...)

	// Assert that it implements the Eqer interface.
	_ compare.Eqer = (*Page)(nil)
	_ compare.Eqer = (*PageOutput)(nil)

	// Assert that it implements the interface needed for related searches.
	_ related.Document = (*Page)(nil)
)

const (
	KindPage = "page"

	// The rest are node types; home page, sections etc.

	KindHome         = "home"
	KindSection      = "section"
	KindTaxonomy     = "taxonomy"
	KindTaxonomyTerm = "taxonomyTerm"

	// Temporary state.
	kindUnknown = "unknown"

	// The following are (currently) temporary nodes,
	// i.e. nodes we create just to render in isolation.
	kindRSS       = "RSS"
	kindSitemap   = "sitemap"
	kindRobotsTXT = "robotsTXT"
	kind404       = "404"

	pageResourceType = "page"
)

type Page struct {
	*pageInit

	// Kind is the discriminator that identifies the different page types
	// in the different page collections. This can, as an example, be used
	// to to filter regular pages, find sections etc.
	// Kind will, for the pages available to the templates, be one of:
	// page, home, section, taxonomy and taxonomyTerm.
	// It is of string type to make it easy to reason about in
	// the templates.
	Kind string

	// Since Hugo 0.18 we got rid of the Node type. So now all pages are ...
	// pages (regular pages, home page, sections etc.).
	// Sections etc. will have child pages. These were earlier placed in .Data.Pages,
	// but can now be more intuitively also be fetched directly from .Pages.
	// This collection will be nil for regular pages.
	Pages Pages

	// Since Hugo 0.32, a Page can have resources such as images and CSS associated
	// with itself. The resource will typically be placed relative to the Page,
	// but templates should use the links (Permalink and RelPermalink)
	// provided by the Resource object.
	Resources resource.Resources

	// This is the raw front matter metadata that is going to be assigned to
	// the Resources above.
	resourcesMetadata []map[string]interface{}

	// translations will contain references to this page in other language
	// if available.
	translations Pages

	// A key that maps to translation(s) of this page. This value is fetched
	// from the page front matter.
	translationKey string

	// Params contains configuration defined in the params section of page frontmatter.
	params map[string]interface{}

	// Content sections
	Content         template.HTML
	Summary         template.HTML
	TableOfContents template.HTML

	Aliases []string

	Images []Image
	Videos []Video

	Truncated bool
	Draft     bool
	Status    string

	PublishDate time.Time
	ExpiryDate  time.Time

	// PageMeta contains page stats such as word count etc.
	PageMeta

	// Markup contains the markup type for the content.
	Markup string

	extension   string
	contentType string
	renderable  bool

	Layout string

	// For npn-renderable pages (see IsRenderable), the content itself
	// is used as template and the template name is stored here.
	selfLayout string

	linkTitle string

	frontmatter []byte

	// rawContent is the raw content read from the content file.
	rawContent []byte

	// workContent is a copy of rawContent that may be mutated during site build.
	workContent []byte

	// whether the content is in a CJK language.
	isCJKLanguage bool

	shortcodeState *shortcodeHandler

	// the content stripped for HTML
	plain      string // TODO should be []byte
	plainWords []string

	// rendering configuration
	renderingConfig *helpers.BlackFriday

	// menus
	pageMenus PageMenus

	Source

	Position `json:"-"`

	GitInfo *gitmap.GitInfo

	// This was added as part of getting the Nodes (taxonomies etc.) to work as
	// Pages in Hugo 0.18.
	// It is deliberately named similar to Section, but not exported (for now).
	// We currently have only one level of section in Hugo, but the page can live
	// any number of levels down the file path.
	// To support taxonomies like /categories/hugo etc. we will need to keep track
	// of that information in a general way.
	// So, sections represents the path to the content, i.e. a content file or a
	// virtual content file in the situations where a taxonomy or a section etc.
	// isn't accomanied by one.
	sections []string

	// Will only be set for sections and regular pages.
	parent *Page

	// When we create paginator pages, we create a copy of the original,
	// but keep track of it here.
	origOnCopy *Page

	// Will only be set for section pages and the home page.
	subSections Pages

	s *Site

	// Pulled over from old Node. TODO(bep) reorg and group (embed)

	Site *SiteInfo `json:"-"`

	title       string
	Description string
	Keywords    []string
	Data        map[string]interface{}

	Date    time.Time
	Lastmod time.Time

	Sitemap Sitemap
	URLPath
	permalink    string
	relPermalink string

	// relative target path without extension and any base path element from the baseURL.
	// This is used to construct paths in the page resources.
	relTargetPathBase string
	// Is set to a forward slashed path if this is a Page resources living in a folder below its owner.
	resourcePath string

	// This is enabled if it is a leaf bundle (the "index.md" type) and it is marked as headless in front matter.
	// Being headless means that
	// 1. The page itself is not rendered to disk
	// 2. It is not available in .Site.Pages etc.
	// 3. But you can get it via .Site.GetPage
	headless bool

	layoutDescriptor output.LayoutDescriptor

	scratch *Scratch

	// It would be tempting to use the language set on the Site, but in they way we do
	// multi-site processing, these values may differ during the initial page processing.
	language *helpers.Language

	lang string

	// The output formats this page will be rendered to.
	outputFormats output.Formats

	// This is the PageOutput that represents the first item in outputFormats.
	// Use with care, as there are potential for inifinite loops.
	mainPageOutput *PageOutput

	targetPathDescriptorPrototype *targetPathDescriptor
}

// SearchKeywords implements the related.Document interface needed for fast page searches.
func (p *Page) SearchKeywords(cfg related.IndexConfig) ([]related.Keyword, error) {

	v, err := p.Param(cfg.Name)
	if err != nil {
		return nil, err
	}

	return cfg.ToKeywords(v)
}

// PubDate is when this page was or will be published.
// NOTE: This is currently used for search only and is not meant to be used
// directly in templates. We need to consolidate the dates in this struct.
// TODO(bep) see https://github.com/gohugoio/hugo/issues/3854
func (p *Page) PubDate() time.Time {
	if !p.PublishDate.IsZero() {
		return p.PublishDate
	}
	return p.Date
}

func (*Page) ResourceType() string {
	return pageResourceType
}

func (p *Page) RSSLink() template.URL {
	f, found := p.outputFormats.GetByName(output.RSSFormat.Name)
	if !found {
		return ""
	}
	return template.URL(newOutputFormat(p, f).Permalink())
}

func (p *Page) createLayoutDescriptor() output.LayoutDescriptor {
	var section string

	switch p.Kind {
	case KindSection:
		// In Hugo 0.22 we introduce nested sections, but we still only
		// use the first level to pick the correct template. This may change in
		// the future.
		section = p.sections[0]
	case KindTaxonomy, KindTaxonomyTerm:
		section = p.s.taxonomiesPluralSingular[p.sections[0]]
	default:
	}

	return output.LayoutDescriptor{
		Kind:    p.Kind,
		Type:    p.Type(),
		Lang:    p.Lang(),
		Layout:  p.Layout,
		Section: section,
	}
}

// pageInit lazy initializes different parts of the page. It is extracted
// into its own type so we can easily create a copy of a given page.
type pageInit struct {
	languageInit        sync.Once
	pageMenusInit       sync.Once
	pageMetaInit        sync.Once
	pageOutputInit      sync.Once
	plainInit           sync.Once
	plainWordsInit      sync.Once
	renderingConfigInit sync.Once
}

// IsNode returns whether this is an item of one of the list types in Hugo,
// i.e. not a regular content page.
func (p *Page) IsNode() bool {
	return p.Kind != KindPage
}

// IsHome returns whether this is the home page.
func (p *Page) IsHome() bool {
	return p.Kind == KindHome
}

// IsSection returns whether this is a section page.
func (p *Page) IsSection() bool {
	return p.Kind == KindSection
}

// IsPage returns whether this is a regular content page.
func (p *Page) IsPage() bool {
	return p.Kind == KindPage
}

type Source struct {
	Frontmatter []byte
	Content     []byte
	source.File
}
type PageMeta struct {
	wordCount      int
	fuzzyWordCount int
	readingTime    int
	Weight         int
}

type Position struct {
	Prev          *Page
	Next          *Page
	PrevInSection *Page
	NextInSection *Page
}

type Pages []*Page

func (ps Pages) String() string {
	return fmt.Sprintf("Pages(%d)", len(ps))
}

func (ps Pages) findPagePosByFilePath(inPath string) int {
	for i, x := range ps {
		if x.Source.Path() == inPath {
			return i
		}
	}
	return -1
}

func (ps Pages) removeFirstIfFound(p *Page) Pages {
	ii := -1
	for i, pp := range ps {
		if pp == p {
			ii = i
			break
		}
	}

	if ii != -1 {
		ps = append(ps[:ii], ps[ii+1:]...)
	}
	return ps
}

func (ps Pages) findFirstPagePosByFilePathPrefix(prefix string) int {
	if prefix == "" {
		return -1
	}
	for i, x := range ps {
		if strings.HasPrefix(x.Source.Path(), prefix) {
			return i
		}
	}
	return -1
}

// findPagePos Given a page, it will find the position in Pages
// will return -1 if not found
func (ps Pages) findPagePos(page *Page) int {
	for i, x := range ps {
		if x.Source.Path() == page.Source.Path() {
			return i
		}
	}
	return -1
}

func (p *Page) createWorkContentCopy() {
	p.workContent = make([]byte, len(p.rawContent))
	copy(p.workContent, p.rawContent)
}

func (p *Page) Plain() string {
	p.initPlain()
	return p.plain
}

func (p *Page) PlainWords() []string {
	p.initPlainWords()
	return p.plainWords
}

func (p *Page) initPlain() {
	p.plainInit.Do(func() {
		p.plain = helpers.StripHTML(string(p.Content))
		return
	})
}

func (p *Page) initPlainWords() {
	p.plainWordsInit.Do(func() {
		p.plainWords = strings.Fields(p.Plain())
		return
	})
}

// Param is a convenience method to do lookups in Page's and Site's Params map,
// in that order.
//
// This method is also implemented on Node and SiteInfo.
func (p *Page) Param(key interface{}) (interface{}, error) {
	keyStr, err := cast.ToStringE(key)
	if err != nil {
		return nil, err
	}

	keyStr = strings.ToLower(keyStr)
	result, _ := p.traverseDirect(keyStr)
	if result != nil {
		return result, nil
	}

	keySegments := strings.Split(keyStr, ".")
	if len(keySegments) == 1 {
		return nil, nil
	}

	return p.traverseNested(keySegments)
}

func (p *Page) traverseDirect(key string) (interface{}, error) {
	keyStr := strings.ToLower(key)
	if val, ok := p.params[keyStr]; ok {
		return val, nil
	}

	return p.Site.Params[keyStr], nil
}

func (p *Page) traverseNested(keySegments []string) (interface{}, error) {
	result := traverse(keySegments, p.params)
	if result != nil {
		return result, nil
	}

	result = traverse(keySegments, p.Site.Params)
	if result != nil {
		return result, nil
	}

	// Didn't find anything, but also no problems.
	return nil, nil
}

func traverse(keys []string, m map[string]interface{}) interface{} {
	// Shift first element off.
	firstKey, rest := keys[0], keys[1:]
	result := m[firstKey]

	// No point in continuing here.
	if result == nil {
		return result
	}

	if len(rest) == 0 {
		// That was the last key.
		return result
	}

	// That was not the last key.
	return traverse(rest, cast.ToStringMap(result))
}

func (p *Page) Author() Author {
	authors := p.Authors()

	for _, author := range authors {
		return author
	}
	return Author{}
}

func (p *Page) Authors() AuthorList {
	authorKeys, ok := p.params["authors"]
	if !ok {
		return AuthorList{}
	}
	authors := authorKeys.([]string)
	if len(authors) < 1 || len(p.Site.Authors) < 1 {
		return AuthorList{}
	}

	al := make(AuthorList)
	for _, author := range authors {
		a, ok := p.Site.Authors[author]
		if ok {
			al[author] = a
		}
	}
	return al
}

func (p *Page) UniqueID() string {
	return p.Source.UniqueID()
}

// for logging
func (p *Page) lineNumRawContentStart() int {
	return bytes.Count(p.frontmatter, []byte("\n")) + 1
}

var (
	internalSummaryDivider = []byte("HUGOMORE42")
)

// replaceDivider replaces the <!--more--> with an internal value and returns
// whether the contentis truncated or not.
// Note: The content slice will be modified if needed.
func replaceDivider(content, from, to []byte) ([]byte, bool) {
	dividerIdx := bytes.Index(content, from)
	if dividerIdx == -1 {
		return content, false
	}

	afterSummary := content[dividerIdx+len(from):]

	// If the raw content has nothing but whitespace after the summary
	// marker then the page shouldn't be marked as truncated.  This check
	// is simplest against the raw content because different markup engines
	// (rst and asciidoc in particular) add div and p elements after the
	// summary marker.
	truncated := bytes.IndexFunc(afterSummary, func(r rune) bool { return !unicode.IsSpace(r) }) != -1

	content = append(content[:dividerIdx], append(to, afterSummary...)...)

	return content, truncated

}

// We have to replace the <!--more--> with something that survives all the
// rendering engines.
func (p *Page) replaceDivider(content []byte) []byte {
	summaryDivider := helpers.SummaryDivider
	// TODO(bep) handle better.
	if p.Ext() == "org" || p.Markup == "org" {
		summaryDivider = []byte("# more")
	}

	replaced, truncated := replaceDivider(content, summaryDivider, internalSummaryDivider)

	p.Truncated = truncated

	return replaced
}

// Returns the page as summary and main if a user defined split is provided.
func (p *Page) setUserDefinedSummaryIfProvided(rawContentCopy []byte) (*summaryContent, error) {

	sc, err := splitUserDefinedSummaryAndContent(p.Markup, rawContentCopy)

	if err != nil {
		return nil, err
	}

	if sc == nil {
		// No divider found
		return nil, nil
	}

	p.Summary = helpers.BytesToHTML(sc.summary)

	return sc, nil
}

// Make this explicit so there is no doubt about what is what.
type summaryContent struct {
	summary []byte
	content []byte
}

func splitUserDefinedSummaryAndContent(markup string, c []byte) (sc *summaryContent, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("summary split failed: %s", r)
		}
	}()

	c = bytes.TrimSpace(c)
	startDivider := bytes.Index(c, internalSummaryDivider)

	if startDivider == -1 {
		return
	}

	endDivider := startDivider + len(internalSummaryDivider)
	endSummary := startDivider

	var (
		startMarkup []byte
		endMarkup   []byte
		addDiv      bool
	)

	switch markup {
	default:
		startMarkup = []byte("<p>")
		endMarkup = []byte("</p>")
	case "asciidoc":
		startMarkup = []byte("<div class=\"paragraph\">")
		endMarkup = []byte("</div>")
	case "rst":
		startMarkup = []byte("<p>")
		endMarkup = []byte("</p>")
		addDiv = true
	}

	// Find the closest end/start markup string to the divider
	fromStart := -1
	fromIdx := bytes.LastIndex(c[:startDivider], startMarkup)
	if fromIdx != -1 {
		fromStart = startDivider - fromIdx - len(startMarkup)
	}
	fromEnd := bytes.Index(c[endDivider:], endMarkup)

	if fromEnd != -1 && fromEnd <= fromStart {
		endSummary = startDivider + fromEnd + len(endMarkup)
	} else if fromStart != -1 && fromEnd != -1 {
		endSummary = startDivider - fromStart - len(startMarkup)
	}

	withoutDivider := bytes.TrimSpace(append(c[:startDivider], c[endDivider:]...))
	var (
		summary []byte
	)

	if len(withoutDivider) > 0 {
		summary = bytes.TrimSpace(withoutDivider[:endSummary])
	}

	if addDiv {
		// For the rst
		summary = append(append([]byte(nil), summary...), []byte("</div>")...)
	}

	if err != nil {
		return
	}

	sc = &summaryContent{
		summary: summary,
		content: withoutDivider,
	}

	return
}

func (p *Page) setAutoSummary() error {
	var summary string
	var truncated bool
	if p.isCJKLanguage {
		summary, truncated = p.s.ContentSpec.TruncateWordsByRune(p.PlainWords())
	} else {
		summary, truncated = p.s.ContentSpec.TruncateWordsToWholeSentence(p.Plain())
	}
	p.Summary = template.HTML(summary)
	p.Truncated = truncated

	return nil
}

func (p *Page) renderContent(content []byte) []byte {
	return p.s.ContentSpec.RenderBytes(&helpers.RenderingContext{
		Content: content, RenderTOC: true, PageFmt: p.determineMarkupType(),
		Cfg:        p.Language(),
		DocumentID: p.UniqueID(), DocumentName: p.Path(),
		Config: p.getRenderingConfig()})
}

func (p *Page) getRenderingConfig() *helpers.BlackFriday {
	p.renderingConfigInit.Do(func() {
		bfParam := p.getParamToLower("blackfriday")
		if bfParam == nil {
			p.renderingConfig = p.s.ContentSpec.BlackFriday
			return
		}
		// Create a copy so we can modify it.
		bf := *p.s.ContentSpec.BlackFriday
		p.renderingConfig = &bf

		if p.Language() == nil {
			panic(fmt.Sprintf("nil language for %s with source lang %s", p.BaseFileName(), p.lang))
		}

		pageParam := cast.ToStringMap(bfParam)
		if err := mapstructure.Decode(pageParam, &p.renderingConfig); err != nil {
			p.s.Log.FATAL.Printf("Failed to get rendering config for %s:\n%s", p.BaseFileName(), err.Error())
		}

	})

	return p.renderingConfig
}

func (s *Site) newPage(filename string) *Page {
	fi := newFileInfo(
		s.SourceSpec,
		s.absContentDir(),
		filename,
		nil,
		bundleNot,
	)
	return s.newPageFromFile(fi)
}

func (s *Site) newPageFromFile(fi *fileInfo) *Page {
	return &Page{
		pageInit:    &pageInit{},
		Kind:        kindFromFileInfo(fi),
		contentType: "",
		Source:      Source{File: fi},
		Keywords:    []string{}, Sitemap: Sitemap{Priority: -1},
		params:       make(map[string]interface{}),
		translations: make(Pages, 0),
		sections:     sectionsFromFile(fi),
		Site:         &s.Info,
		s:            s,
	}
}

func (p *Page) IsRenderable() bool {
	return p.renderable
}

func (p *Page) Type() string {
	if p.contentType != "" {
		return p.contentType
	}

	if x := p.Section(); x != "" {
		return x
	}

	return "page"
}

// Section returns the first path element below the content root. Note that
// since Hugo 0.22 we support nested sections, but this will always be the first
// element of any nested path.
func (p *Page) Section() string {
	if p.Kind == KindSection || p.Kind == KindTaxonomy || p.Kind == KindTaxonomyTerm {
		return p.sections[0]
	}
	return p.Source.Section()
}

func (s *Site) NewPageFrom(buf io.Reader, name string) (*Page, error) {
	p, err := s.NewPage(name)
	if err != nil {
		return p, err
	}
	_, err = p.ReadFrom(buf)

	return p, err
}

func (s *Site) NewPage(name string) (*Page, error) {
	if len(name) == 0 {
		return nil, errors.New("Zero length page name")
	}

	// Create new page
	p := s.newPage(name)
	p.s = s
	p.Site = &s.Info

	return p, nil
}

func (p *Page) ReadFrom(buf io.Reader) (int64, error) {
	// Parse for metadata & body
	if err := p.parse(buf); err != nil {
		p.s.Log.ERROR.Print(err)
		return 0, err
	}

	return int64(len(p.rawContent)), nil
}

func (p *Page) WordCount() int {
	p.analyzePage()
	return p.wordCount
}

func (p *Page) ReadingTime() int {
	p.analyzePage()
	return p.readingTime
}

func (p *Page) FuzzyWordCount() int {
	p.analyzePage()
	return p.fuzzyWordCount
}

func (p *Page) analyzePage() {
	p.pageMetaInit.Do(func() {
		if p.isCJKLanguage {
			p.wordCount = 0
			for _, word := range p.PlainWords() {
				runeCount := utf8.RuneCountInString(word)
				if len(word) == runeCount {
					p.wordCount++
				} else {
					p.wordCount += runeCount
				}
			}
		} else {
			p.wordCount = helpers.TotalWords(p.Plain())
		}

		// TODO(bep) is set in a test. Fix that.
		if p.fuzzyWordCount == 0 {
			p.fuzzyWordCount = (p.wordCount + 100) / 100 * 100
		}

		if p.isCJKLanguage {
			p.readingTime = (p.wordCount + 500) / 501
		} else {
			p.readingTime = (p.wordCount + 212) / 213
		}
	})
}

// HasShortcode return whether the page has a shortcode with the given name.
// This method is mainly motivated with the Hugo Docs site's need for a list
// of pages with the `todo` shortcode in it.
func (p *Page) HasShortcode(name string) bool {
	if p.shortcodeState == nil {
		return false
	}

	return p.shortcodeState.nameSet[name]
}

// AllTranslations returns all translations, including the current Page.
func (p *Page) AllTranslations() Pages {
	return p.translations
}

// IsTranslated returns whether this content file is translated to
// other language(s).
func (p *Page) IsTranslated() bool {
	return len(p.translations) > 1
}

// Translations returns the translations excluding the current Page.
func (p *Page) Translations() Pages {
	translations := make(Pages, 0)
	for _, t := range p.translations {
		if t.Lang() != p.Lang() {
			translations = append(translations, t)
		}
	}
	return translations
}

// TranslationKey returns the key used to map language translations of this page.
// It will use the translationKey set in front matter if set, or the content path and
// filename (excluding any language code and extension), e.g. "about/index".
// The Page Kind is always prepended.
func (p *Page) TranslationKey() string {
	if p.translationKey != "" {
		return p.Kind + "/" + p.translationKey
	}

	if p.IsNode() {
		return path.Join(p.Kind, path.Join(p.sections...), p.TranslationBaseName())
	}

	return path.Join(p.Kind, filepath.ToSlash(p.Dir()), p.TranslationBaseName())
}

func (p *Page) LinkTitle() string {
	if len(p.linkTitle) > 0 {
		return p.linkTitle
	}
	return p.title
}

func (p *Page) shouldBuild() bool {
	return shouldBuild(p.s.BuildFuture, p.s.BuildExpired,
		p.s.BuildDrafts, p.Draft, p.PublishDate, p.ExpiryDate)
}

func shouldBuild(buildFuture bool, buildExpired bool, buildDrafts bool, Draft bool,
	publishDate time.Time, expiryDate time.Time) bool {
	if !(buildDrafts || !Draft) {
		return false
	}
	if !buildFuture && !publishDate.IsZero() && publishDate.After(time.Now()) {
		return false
	}
	if !buildExpired && !expiryDate.IsZero() && expiryDate.Before(time.Now()) {
		return false
	}
	return true
}

func (p *Page) IsDraft() bool {
	return p.Draft
}

func (p *Page) IsFuture() bool {
	if p.PublishDate.IsZero() {
		return false
	}
	return p.PublishDate.After(time.Now())
}

func (p *Page) IsExpired() bool {
	if p.ExpiryDate.IsZero() {
		return false
	}
	return p.ExpiryDate.Before(time.Now())
}

func (p *Page) URL() string {

	if p.IsPage() && p.URLPath.URL != "" {
		// This is the url set in front matter
		return p.URLPath.URL
	}
	// Fall back to the relative permalink.
	u := p.RelPermalink()
	return u
}

// Permalink returns the absolute URL to this Page.
func (p *Page) Permalink() string {
	if p.headless {
		return ""
	}
	return p.permalink
}

// RelPermalink gets a URL to the resource relative to the host.
func (p *Page) RelPermalink() string {
	if p.headless {
		return ""
	}
	return p.relPermalink
}

// See resource.Resource
// This value is used, by default, in Resources.ByPrefix etc.
func (p *Page) Name() string {
	if p.resourcePath != "" {
		return p.resourcePath
	}
	return p.title
}

func (p *Page) Title() string {
	return p.title
}

func (p *Page) Params() map[string]interface{} {
	return p.params
}

func (p *Page) subResourceTargetPathFactory(base string) string {
	return path.Join(p.relTargetPathBase, base)
}

func (p *Page) prepareForRender(cfg *BuildCfg) error {
	s := p.s

	if !p.shouldRenderTo(s.rc.Format) {
		// No need to prepare
		return nil
	}

	var shortcodeUpdate bool
	if p.shortcodeState != nil {
		shortcodeUpdate = p.shortcodeState.updateDelta()
	}

	if !shortcodeUpdate && !cfg.whatChanged.other {
		// No need to process it again.
		return nil
	}

	// If we got this far it means that this is either a new Page pointer
	// or a template or similar has changed so wee need to do a rerendering
	// of the shortcodes etc.

	// If in watch mode or if we have multiple output formats,
	// we need to keep the original so we can
	// potentially repeat this process on rebuild.
	needsACopy := p.s.running() || len(p.outputFormats) > 1
	var workContentCopy []byte
	if needsACopy {
		workContentCopy = make([]byte, len(p.workContent))
		copy(workContentCopy, p.workContent)
	} else {
		// Just reuse the same slice.
		workContentCopy = p.workContent
	}

	if p.Markup == "markdown" {
		tmpContent, tmpTableOfContents := helpers.ExtractTOC(workContentCopy)
		p.TableOfContents = helpers.BytesToHTML(tmpTableOfContents)
		workContentCopy = tmpContent
	}

	var err error
	if workContentCopy, err = handleShortcodes(p, workContentCopy); err != nil {
		s.Log.ERROR.Printf("Failed to handle shortcodes for page %s: %s", p.BaseFileName(), err)
	}

	if p.Markup != "html" {

		// Now we know enough to create a summary of the page and count some words
		summaryContent, err := p.setUserDefinedSummaryIfProvided(workContentCopy)

		if err != nil {
			s.Log.ERROR.Printf("Failed to set user defined summary for page %q: %s", p.Path(), err)
		} else if summaryContent != nil {
			workContentCopy = summaryContent.content
		}

		p.Content = helpers.BytesToHTML(workContentCopy)

		if summaryContent == nil {
			if err := p.setAutoSummary(); err != nil {
				s.Log.ERROR.Printf("Failed to set user auto summary for page %q: %s", p.pathOrTitle(), err)
			}
		}

	} else {
		p.Content = helpers.BytesToHTML(workContentCopy)
	}

	//analyze for raw stats
	p.analyzePage()

	// Handle bundled pages.
	for _, r := range p.Resources.ByType(pageResourceType) {
		p.s.PathSpec.ProcessingStats.Incr(&p.s.PathSpec.ProcessingStats.Pages)
		bp := r.(*Page)
		if err := bp.prepareForRender(cfg); err != nil {
			s.Log.ERROR.Printf("Failed to prepare bundled page %q for render: %s", bp.BaseFileName(), err)
		}
	}

	return nil
}

var ErrHasDraftAndPublished = errors.New("both draft and published parameters were found in page's frontmatter")

func (p *Page) update(frontmatter map[string]interface{}) error {
	if frontmatter == nil {
		return errors.New("missing frontmatter data")
	}
	// Needed for case insensitive fetching of params values
	helpers.ToLowerMap(frontmatter)

	var modified time.Time

	var err error
	var draft, published, isCJKLanguage *bool
	for k, v := range frontmatter {
		loki := strings.ToLower(k)
		switch loki {
		case "title":
			p.title = cast.ToString(v)
			p.params[loki] = p.title
		case "linktitle":
			p.linkTitle = cast.ToString(v)
			p.params[loki] = p.linkTitle
		case "description":
			p.Description = cast.ToString(v)
			p.params[loki] = p.Description
		case "slug":
			p.Slug = cast.ToString(v)
			p.params[loki] = p.Slug
		case "url":
			if url := cast.ToString(v); strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
				return fmt.Errorf("Only relative URLs are supported, %v provided", url)
			}
			p.URLPath.URL = cast.ToString(v)
			p.URLPath.frontMatterURL = p.URLPath.URL
			p.params[loki] = p.URLPath.URL
		case "type":
			p.contentType = cast.ToString(v)
			p.params[loki] = p.contentType
		case "extension", "ext":
			p.extension = cast.ToString(v)
			p.params[loki] = p.extension
		case "keywords":
			p.Keywords = cast.ToStringSlice(v)
			p.params[loki] = p.Keywords
		case "date":
			p.Date, err = cast.ToTimeE(v)
			if err != nil {
				p.s.Log.ERROR.Printf("Failed to parse date '%v' in page %s", v, p.File.Path())
			}
			p.params[loki] = p.Date
		case "headless":
			// For now, only the leaf bundles ("index.md") can be headless (i.e. produce no output).
			// We may expand on this in the future, but that gets more complex pretty fast.
			if p.TranslationBaseName() == "index" {
				p.headless = cast.ToBool(v)
			}
			p.params[loki] = p.headless
		case "lastmod":
			p.Lastmod, err = cast.ToTimeE(v)
			if err != nil {
				p.s.Log.ERROR.Printf("Failed to parse lastmod '%v' in page %s", v, p.File.Path())
			}
		case "modified":
			vv, err := cast.ToTimeE(v)
			if err == nil {
				p.params[loki] = vv
				modified = vv
			} else {
				p.params[loki] = cast.ToString(v)
			}
		case "outputs":
			o := cast.ToStringSlice(v)
			if len(o) > 0 {
				// Output formats are exlicitly set in front matter, use those.
				outFormats, err := p.s.outputFormatsConfig.GetByNames(o...)

				if err != nil {
					p.s.Log.ERROR.Printf("Failed to resolve output formats: %s", err)
				} else {
					p.outputFormats = outFormats
					p.params[loki] = outFormats
				}

			}
		case "publishdate", "pubdate":
			p.PublishDate, err = cast.ToTimeE(v)
			if err != nil {
				p.s.Log.ERROR.Printf("Failed to parse publishdate '%v' in page %s", v, p.File.Path())
			}
			p.params[loki] = p.PublishDate
		case "expirydate", "unpublishdate":
			p.ExpiryDate, err = cast.ToTimeE(v)
			if err != nil {
				p.s.Log.ERROR.Printf("Failed to parse expirydate '%v' in page %s", v, p.File.Path())
			}
		case "draft":
			draft = new(bool)
			*draft = cast.ToBool(v)
		case "published": // Intentionally undocumented
			vv, err := cast.ToBoolE(v)
			if err == nil {
				published = &vv
			} else {
				// Some sites use this as the publishdate
				vv, err := cast.ToTimeE(v)
				if err == nil {
					p.PublishDate = vv
					p.params[loki] = p.PublishDate
				} else {
					p.params[loki] = cast.ToString(v)
				}
			}
		case "layout":
			p.Layout = cast.ToString(v)
			p.params[loki] = p.Layout
		case "markup":
			p.Markup = cast.ToString(v)
			p.params[loki] = p.Markup
		case "weight":
			p.Weight = cast.ToInt(v)
			p.params[loki] = p.Weight
		case "aliases":
			p.Aliases = cast.ToStringSlice(v)
			for _, alias := range p.Aliases {
				if strings.HasPrefix(alias, "http://") || strings.HasPrefix(alias, "https://") {
					return fmt.Errorf("Only relative aliases are supported, %v provided", alias)
				}
			}
			p.params[loki] = p.Aliases
		case "status":
			p.Status = cast.ToString(v)
			p.params[loki] = p.Status
		case "sitemap":
			p.Sitemap = parseSitemap(cast.ToStringMap(v))
			p.params[loki] = p.Sitemap
		case "iscjklanguage":
			isCJKLanguage = new(bool)
			*isCJKLanguage = cast.ToBool(v)
		case "translationkey":
			p.translationKey = cast.ToString(v)
			p.params[loki] = p.translationKey
		case "resources":
			var resources []map[string]interface{}
			handled := true

			switch vv := v.(type) {
			case []map[interface{}]interface{}:
				for _, vvv := range vv {
					resources = append(resources, cast.ToStringMap(vvv))
				}
			case []map[string]interface{}:
				for _, vvv := range vv {
					resources = append(resources, vvv)
				}
			case []interface{}:
				for _, vvv := range vv {
					switch vvvv := vvv.(type) {
					case map[interface{}]interface{}:
						resources = append(resources, cast.ToStringMap(vvvv))
					case map[string]interface{}:
						resources = append(resources, vvvv)
					}
				}
			default:
				handled = false
			}

			if handled {
				p.params[loki] = resources
				p.resourcesMetadata = resources
				break
			}
			fallthrough

		default:
			// If not one of the explicit values, store in Params
			switch vv := v.(type) {
			case bool:
				p.params[loki] = vv
			case string:
				p.params[loki] = vv
			case int64, int32, int16, int8, int:
				p.params[loki] = vv
			case float64, float32:
				p.params[loki] = vv
			case time.Time:
				p.params[loki] = vv
			default: // handle array of strings as well
				switch vvv := vv.(type) {
				case []interface{}:
					if len(vvv) > 0 {
						switch vvv[0].(type) {
						case map[interface{}]interface{}: // Proper parsing structured array from YAML based FrontMatter
							p.params[loki] = vvv
						case map[string]interface{}: // Proper parsing structured array from JSON based FrontMatter
							p.params[loki] = vvv
						case []interface{}:
							p.params[loki] = vvv
						default:
							a := make([]string, len(vvv))
							for i, u := range vvv {
								a[i] = cast.ToString(u)
							}

							p.params[loki] = a
						}
					} else {
						p.params[loki] = []string{}
					}
				default:
					p.params[loki] = vv
				}
			}
		}
	}

	if draft != nil && published != nil {
		p.Draft = *draft
		p.s.Log.ERROR.Printf("page %s has both draft and published settings in its frontmatter. Using draft.", p.File.Path())
		return ErrHasDraftAndPublished
	} else if draft != nil {
		p.Draft = *draft
	} else if published != nil {
		p.Draft = !*published
	}
	p.params["draft"] = p.Draft

	if p.Date.IsZero() {
		p.Date = p.PublishDate
	}

	if p.PublishDate.IsZero() {
		p.PublishDate = p.Date
	}

	if p.Date.IsZero() && p.s.Cfg.GetBool("useModTimeAsFallback") {
		p.Date = p.Source.FileInfo().ModTime()
	}

	if p.Lastmod.IsZero() {
		if !modified.IsZero() {
			p.Lastmod = modified
		} else {
			p.Lastmod = p.Date
		}

	}

	p.params["date"] = p.Date
	p.params["lastmod"] = p.Lastmod
	p.params["publishdate"] = p.PublishDate
	p.params["expirydate"] = p.ExpiryDate

	if isCJKLanguage != nil {
		p.isCJKLanguage = *isCJKLanguage
	} else if p.s.Cfg.GetBool("hasCJKLanguage") {
		if cjk.Match(p.rawContent) {
			p.isCJKLanguage = true
		} else {
			p.isCJKLanguage = false
		}
	}
	p.params["iscjklanguage"] = p.isCJKLanguage

	return nil
}

func (p *Page) GetParam(key string) interface{} {
	return p.getParam(key, false)
}

func (p *Page) getParamToLower(key string) interface{} {
	return p.getParam(key, true)
}

func (p *Page) getParam(key string, stringToLower bool) interface{} {
	v := p.params[strings.ToLower(key)]

	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case bool:
		return val
	case string:
		if stringToLower {
			return strings.ToLower(val)
		}
		return val
	case int64, int32, int16, int8, int:
		return cast.ToInt(v)
	case float64, float32:
		return cast.ToFloat64(v)
	case time.Time:
		return val
	case []string:
		if stringToLower {
			return helpers.SliceToLower(val)
		}
		return v
	case map[string]interface{}: // JSON and TOML
		return v
	case map[interface{}]interface{}: // YAML
		return v
	}

	p.s.Log.ERROR.Printf("GetParam(\"%s\"): Unknown type %s\n", key, reflect.TypeOf(v))
	return nil
}

func (p *Page) HasMenuCurrent(menuID string, me *MenuEntry) bool {

	sectionPagesMenu := p.Site.sectionPagesMenu

	// page is labeled as "shadow-member" of the menu with the same identifier as the section
	if sectionPagesMenu != "" {
		section := p.Section()

		if section != "" && sectionPagesMenu == menuID && section == me.Identifier {
			return true
		}
	}

	if !me.HasChildren() {
		return false
	}

	menus := p.Menus()

	if m, ok := menus[menuID]; ok {

		for _, child := range me.Children {
			if child.IsEqual(m) {
				return true
			}
			if p.HasMenuCurrent(menuID, child) {
				return true
			}
		}

	}

	if p.IsPage() {
		return false
	}

	// The following logic is kept from back when Hugo had both Page and Node types.
	// TODO(bep) consolidate / clean
	nme := MenuEntry{Page: p, Name: p.title, URL: p.URL()}

	for _, child := range me.Children {
		if nme.IsSameResource(child) {
			return true
		}
		if p.HasMenuCurrent(menuID, child) {
			return true
		}
	}

	return false

}

func (p *Page) IsMenuCurrent(menuID string, inme *MenuEntry) bool {

	menus := p.Menus()

	if me, ok := menus[menuID]; ok {
		if me.IsEqual(inme) {
			return true
		}
	}

	if p.IsPage() {
		return false
	}

	// The following logic is kept from back when Hugo had both Page and Node types.
	// TODO(bep) consolidate / clean
	me := MenuEntry{Page: p, Name: p.title, URL: p.URL()}

	if !me.IsSameResource(inme) {
		return false
	}

	// this resource may be included in several menus
	// search for it to make sure that it is in the menu with the given menuId
	if menu, ok := (*p.Site.Menus)[menuID]; ok {
		for _, menuEntry := range *menu {
			if menuEntry.IsSameResource(inme) {
				return true
			}

			descendantFound := p.isSameAsDescendantMenu(inme, menuEntry)
			if descendantFound {
				return descendantFound
			}

		}
	}

	return false
}

func (p *Page) isSameAsDescendantMenu(inme *MenuEntry, parent *MenuEntry) bool {
	if parent.HasChildren() {
		for _, child := range parent.Children {
			if child.IsSameResource(inme) {
				return true
			}
			descendantFound := p.isSameAsDescendantMenu(inme, child)
			if descendantFound {
				return descendantFound
			}
		}
	}
	return false
}

func (p *Page) Menus() PageMenus {
	p.pageMenusInit.Do(func() {
		p.pageMenus = PageMenus{}

		if ms, ok := p.params["menu"]; ok {
			link := p.RelPermalink()

			me := MenuEntry{Page: p, Name: p.LinkTitle(), Weight: p.Weight, URL: link}

			// Could be the name of the menu to attach it to
			mname, err := cast.ToStringE(ms)

			if err == nil {
				me.Menu = mname
				p.pageMenus[mname] = &me
				return
			}

			// Could be a slice of strings
			mnames, err := cast.ToStringSliceE(ms)

			if err == nil {
				for _, mname := range mnames {
					me.Menu = mname
					p.pageMenus[mname] = &me
				}
				return
			}

			// Could be a structured menu entry
			menus, err := cast.ToStringMapE(ms)

			if err != nil {
				p.s.Log.ERROR.Printf("unable to process menus for %q\n", p.title)
			}

			for name, menu := range menus {
				menuEntry := MenuEntry{Page: p, Name: p.LinkTitle(), URL: link, Weight: p.Weight, Menu: name}
				if menu != nil {
					p.s.Log.DEBUG.Printf("found menu: %q, in %q\n", name, p.title)
					ime, err := cast.ToStringMapE(menu)
					if err != nil {
						p.s.Log.ERROR.Printf("unable to process menus for %q: %s", p.title, err)
					}

					menuEntry.marshallMap(ime)
				}
				p.pageMenus[name] = &menuEntry

			}
		}
	})

	return p.pageMenus
}

func (p *Page) shouldRenderTo(f output.Format) bool {
	_, found := p.outputFormats.GetByName(f.Name)
	return found
}

func (p *Page) determineMarkupType() string {
	// Try markup explicitly set in the frontmatter
	p.Markup = helpers.GuessType(p.Markup)
	if p.Markup == "unknown" {
		// Fall back to file extension (might also return "unknown")
		p.Markup = helpers.GuessType(p.Source.Ext())
	}

	return p.Markup
}

func (p *Page) parse(reader io.Reader) error {
	psr, err := parser.ReadFrom(reader)
	if err != nil {
		return err
	}

	p.renderable = psr.IsRenderable()
	p.frontmatter = psr.FrontMatter()
	p.rawContent = psr.Content()
	p.lang = p.Source.File.Lang()

	meta, err := psr.Metadata()
	if err != nil {
		return fmt.Errorf("failed to parse page metadata for %q: %s", p.File.Path(), err)
	}
	if meta == nil {
		// missing frontmatter equivalent to empty frontmatter
		meta = map[string]interface{}{}
	}

	return p.update(meta)
}

func (p *Page) RawContent() string {
	return string(p.rawContent)
}

func (p *Page) SetSourceContent(content []byte) {
	p.Source.Content = content
}

func (p *Page) SetSourceMetaData(in interface{}, mark rune) (err error) {
	// See https://github.com/gohugoio/hugo/issues/2458
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("error from marshal: %v", r)
			}
		}
	}()

	buf := bp.GetBuffer()
	defer bp.PutBuffer(buf)

	err = parser.InterfaceToFrontMatter(in, mark, buf)
	if err != nil {
		return
	}

	_, err = buf.WriteRune('\n')
	if err != nil {
		return
	}

	p.Source.Frontmatter = buf.Bytes()

	return
}

func (p *Page) SafeSaveSourceAs(path string) error {
	return p.saveSourceAs(path, true)
}

func (p *Page) SaveSourceAs(path string) error {
	return p.saveSourceAs(path, false)
}

func (p *Page) saveSourceAs(path string, safe bool) error {
	b := bp.GetBuffer()
	defer bp.PutBuffer(b)

	b.Write(p.Source.Frontmatter)
	b.Write(p.Source.Content)

	bc := make([]byte, b.Len(), b.Len())
	copy(bc, b.Bytes())

	return p.saveSource(bc, path, safe)
}

func (p *Page) saveSource(by []byte, inpath string, safe bool) (err error) {
	if !filepath.IsAbs(inpath) {
		inpath = p.s.PathSpec.AbsPathify(inpath)
	}
	p.s.Log.INFO.Println("creating", inpath)
	if safe {
		err = helpers.SafeWriteToDisk(inpath, bytes.NewReader(by), p.s.Fs.Source)
	} else {
		err = helpers.WriteToDisk(inpath, bytes.NewReader(by), p.s.Fs.Source)
	}
	if err != nil {
		return
	}
	return nil
}

func (p *Page) SaveSource() error {
	return p.SaveSourceAs(p.FullFilePath())
}

func (p *Page) processShortcodes() error {
	p.shortcodeState = newShortcodeHandler(p)
	tmpContent, err := p.shortcodeState.extractShortcodes(string(p.workContent), p)
	if err != nil {
		return err
	}
	p.workContent = []byte(tmpContent)

	return nil

}

func (p *Page) FullFilePath() string {
	return filepath.Join(p.Dir(), p.LogicalName())
}

// Pre render prepare steps

func (p *Page) prepareLayouts() error {
	// TODO(bep): Check the IsRenderable logic.
	if p.Kind == KindPage {
		if !p.IsRenderable() {
			self := "__" + p.UniqueID()
			err := p.s.TemplateHandler().AddLateTemplate(self, string(p.Content))
			if err != nil {
				return err
			}
			p.selfLayout = self
		}
	}

	return nil
}

func (p *Page) prepareData(s *Site) error {
	if p.Kind != KindSection {
		var pages Pages
		p.Data = make(map[string]interface{})

		switch p.Kind {
		case KindPage:
		case KindHome:
			pages = s.RegularPages
		case KindTaxonomy:
			plural := p.sections[0]
			term := p.sections[1]

			if s.Info.preserveTaxonomyNames {
				if v, ok := s.taxonomiesOrigKey[fmt.Sprintf("%s-%s", plural, term)]; ok {
					term = v
				}
			}

			singular := s.taxonomiesPluralSingular[plural]
			taxonomy := s.Taxonomies[plural].Get(term)

			p.Data[singular] = taxonomy
			p.Data["Singular"] = singular
			p.Data["Plural"] = plural
			p.Data["Term"] = term
			pages = taxonomy.Pages()
		case KindTaxonomyTerm:
			plural := p.sections[0]
			singular := s.taxonomiesPluralSingular[plural]

			p.Data["Singular"] = singular
			p.Data["Plural"] = plural
			p.Data["Terms"] = s.Taxonomies[plural]
			// keep the following just for legacy reasons
			p.Data["OrderedIndex"] = p.Data["Terms"]
			p.Data["Index"] = p.Data["Terms"]

			// A list of all KindTaxonomy pages with matching plural
			for _, p := range s.findPagesByKind(KindTaxonomy) {
				if p.sections[0] == plural {
					pages = append(pages, p)
				}
			}
		}

		p.Data["Pages"] = pages
		p.Pages = pages
	}

	// Now we know enough to set missing dates on home page etc.
	p.updatePageDates()

	return nil
}

func (p *Page) updatePageDates() {
	// TODO(bep) there is a potential issue with page sorting for home pages
	// etc. without front matter dates set, but let us wrap the head around
	// that in another time.
	if !p.IsNode() {
		return
	}

	if !p.Date.IsZero() {
		if p.Lastmod.IsZero() {
			p.Lastmod = p.Date
		}
		return
	} else if !p.Lastmod.IsZero() {
		if p.Date.IsZero() {
			p.Date = p.Lastmod
		}
		return
	}

	// Set it to the first non Zero date in children
	var foundDate, foundLastMod bool

	for _, child := range p.Pages {
		if !child.Date.IsZero() {
			p.Date = child.Date
			foundDate = true
		}
		if !child.Lastmod.IsZero() {
			p.Lastmod = child.Lastmod
			foundLastMod = true
		}

		if foundDate && foundLastMod {
			break
		}
	}
}

// copy creates a copy of this page with the lazy sync.Once vars reset
// so they will be evaluated again, for word count calculations etc.
func (p *Page) copy() *Page {
	c := *p
	c.pageInit = &pageInit{}
	return &c
}

func (p *Page) Hugo() *HugoInfo {
	return hugoInfo
}

func (p *Page) Ref(refs ...string) (string, error) {
	if len(refs) == 0 {
		return "", nil
	}
	if len(refs) > 1 {
		return p.Site.Ref(refs[0], nil, refs[1])
	}
	return p.Site.Ref(refs[0], nil)
}

func (p *Page) RelRef(refs ...string) (string, error) {
	if len(refs) == 0 {
		return "", nil
	}
	if len(refs) > 1 {
		return p.Site.RelRef(refs[0], nil, refs[1])
	}
	return p.Site.RelRef(refs[0], nil)
}

func (p *Page) String() string {
	return fmt.Sprintf("Page(%q)", p.title)
}

type URLPath struct {
	URL            string
	frontMatterURL string
	Permalink      string
	Slug           string
	Section        string
}

// Scratch returns the writable context associated with this Page.
func (p *Page) Scratch() *Scratch {
	if p.scratch == nil {
		p.scratch = newScratch()
	}
	return p.scratch
}

func (p *Page) Language() *helpers.Language {
	p.initLanguage()
	return p.language
}

func (p *Page) Lang() string {
	// When set, Language can be different from lang in the case where there is a
	// content file (doc.sv.md) with language indicator, but there is no language
	// config for that language. Then the language will fall back on the site default.
	if p.Language() != nil {
		return p.Language().Lang
	}
	return p.lang
}

func (p *Page) isNewTranslation(candidate *Page) bool {

	if p.Kind != candidate.Kind {
		return false
	}

	if p.Kind == KindPage || p.Kind == kindUnknown {
		panic("Node type not currently supported for this op")
	}

	// At this point, we know that this is a traditional Node (home page, section, taxonomy)
	// It represents the same node, but different language, if the sections is the same.
	if len(p.sections) != len(candidate.sections) {
		return false
	}

	for i := 0; i < len(p.sections); i++ {
		if p.sections[i] != candidate.sections[i] {
			return false
		}
	}

	// Finally check that it is not already added.
	for _, translation := range p.translations {
		if candidate == translation {
			return false
		}
	}

	return true

}

func (p *Page) shouldAddLanguagePrefix() bool {
	if !p.Site.IsMultiLingual() {
		return false
	}

	if p.s.owner.IsMultihost() {
		return true
	}

	if p.Lang() == "" {
		return false
	}

	if !p.Site.defaultContentLanguageInSubdir && p.Lang() == p.Site.multilingual.DefaultLang.Lang {
		return false
	}

	return true
}

func (p *Page) initLanguage() {
	p.languageInit.Do(func() {
		if p.language != nil {
			return
		}

		ml := p.Site.multilingual
		if ml == nil {
			panic("Multilanguage not set")
		}
		if p.lang == "" {
			p.lang = ml.DefaultLang.Lang
			p.language = ml.DefaultLang
			return
		}

		language := ml.Language(p.lang)

		if language == nil {
			// It can be a file named stefano.chiodino.md.
			p.s.Log.WARN.Printf("Page language (if it is that) not found in multilang setup: %s.", p.lang)
			language = ml.DefaultLang
		}

		p.language = language

	})
}

func (p *Page) LanguagePrefix() string {
	return p.Site.LanguagePrefix
}

func (p *Page) addLangPathPrefixIfFlagSet(outfile string, should bool) string {
	if helpers.IsAbsURL(outfile) {
		return outfile
	}

	if !should {
		return outfile
	}

	hadSlashSuffix := strings.HasSuffix(outfile, "/")

	outfile = "/" + path.Join(p.Lang(), outfile)
	if hadSlashSuffix {
		outfile += "/"
	}
	return outfile
}

func sectionsFromFile(fi *fileInfo) []string {
	dirname := fi.Dir()
	dirname = strings.Trim(dirname, helpers.FilePathSeparator)
	if dirname == "" {
		return nil
	}
	parts := strings.Split(dirname, helpers.FilePathSeparator)

	if fi.bundleTp == bundleLeaf && len(parts) > 0 {
		// my-section/mybundle/index.md => my-section
		return parts[:len(parts)-1]
	}

	return parts
}

func kindFromFileInfo(fi *fileInfo) string {
	if fi.TranslationBaseName() == "_index" {
		if fi.Dir() == "" {
			return KindHome
		}
		// Could be index for section, taxonomy, taxonomy term
		// We don't know enough yet to determine which
		return kindUnknown
	}
	return KindPage
}

func (p *Page) setValuesForKind(s *Site) {
	if p.Kind == kindUnknown {
		// This is either a taxonomy list, taxonomy term or a section
		nodeType := s.kindFromSections(p.sections)

		if nodeType == kindUnknown {
			panic(fmt.Sprintf("Unable to determine page kind from %q", p.sections))
		}

		p.Kind = nodeType
	}

	switch p.Kind {
	case KindHome:
		p.URLPath.URL = "/"
	case KindPage:
	default:
		if p.URLPath.URL == "" {
			p.URLPath.URL = "/" + path.Join(p.sections...) + "/"
		}
	}
}

// Used in error logs.
func (p *Page) pathOrTitle() string {
	if p.Path() != "" {
		return p.Path()
	}
	return p.title
}

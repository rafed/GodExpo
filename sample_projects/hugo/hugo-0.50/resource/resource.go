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

package resource

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gohugoio/hugo/output"
	"github.com/gohugoio/hugo/tpl"
	"github.com/pkg/errors"

	"github.com/gohugoio/hugo/common/collections"
	"github.com/gohugoio/hugo/common/hugio"
	"github.com/gohugoio/hugo/common/loggers"

	"github.com/spf13/afero"

	"github.com/gobwas/glob"
	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/media"
	"github.com/gohugoio/hugo/source"
)

var (
	_ ContentResource         = (*genericResource)(nil)
	_ ReadSeekCloserResource  = (*genericResource)(nil)
	_ Resource                = (*genericResource)(nil)
	_ Source                  = (*genericResource)(nil)
	_ Cloner                  = (*genericResource)(nil)
	_ ResourcesLanguageMerger = (*Resources)(nil)
	_ permalinker             = (*genericResource)(nil)
	_ collections.Slicer      = (*genericResource)(nil)
)

var noData = make(map[string]interface{})

// Source is an internal template and not meant for use in the templates. It
// may change without notice.
type Source interface {
	Publish() error
}

type permalinker interface {
	relPermalinkFor(target string) string
	permalinkFor(target string) string
	relTargetPathsFor(target string) []string
	relTargetPaths() []string
	targetPath() string
}

// Cloner is an internal template and not meant for use in the templates. It
// may change without notice.
type Cloner interface {
	WithNewBase(base string) Resource
}

// Resource represents a linkable resource, i.e. a content page, image etc.
type Resource interface {
	// Permalink represents the absolute link to this resource.
	Permalink() string

	// RelPermalink represents the host relative link to this resource.
	RelPermalink() string

	// ResourceType is the resource type. For most file types, this is the main
	// part of the MIME type, e.g. "image", "application", "text" etc.
	// For content pages, this value is "page".
	ResourceType() string

	// MediaType is this resource's MIME type.
	MediaType() media.Type

	// Name is the logical name of this resource. This can be set in the front matter
	// metadata for this resource. If not set, Hugo will assign a value.
	// This will in most cases be the base filename.
	// So, for the image "/some/path/sunset.jpg" this will be "sunset.jpg".
	// The value returned by this method will be used in the GetByPrefix and ByPrefix methods
	// on Resources.
	Name() string

	// Title returns the title if set in front matter. For content pages, this will be the expected value.
	Title() string

	// Resource specific data set by Hugo.
	// One example would be.Data.Digest for fingerprinted resources.
	Data() interface{}

	// Params set in front matter for this resource.
	Params() map[string]interface{}
}

// ResourcesLanguageMerger describes an interface for merging resources from a
// different language.
type ResourcesLanguageMerger interface {
	MergeByLanguage(other Resources) Resources
	// Needed for integration with the tpl package.
	MergeByLanguageInterface(other interface{}) (interface{}, error)
}

type translatedResource interface {
	TranslationKey() string
}

// ContentResource represents a Resource that provides a way to get to its content.
// Most Resource types in Hugo implements this interface, including Page.
// This should be used with care, as it will read the file content into memory, but it
// should be cached as effectively as possible by the implementation.
type ContentResource interface {
	Resource

	// Content returns this resource's content. It will be equivalent to reading the content
	// that RelPermalink points to in the published folder.
	// The return type will be contextual, and should be what you would expect:
	// * Page: template.HTML
	// * JSON: String
	// * Etc.
	Content() (interface{}, error)
}

// OpenReadSeekCloser allows setting some other way (than reading from a filesystem)
// to open or create a ReadSeekCloser.
type OpenReadSeekCloser func() (hugio.ReadSeekCloser, error)

// ReadSeekCloserResource is a Resource that supports loading its content.
type ReadSeekCloserResource interface {
	Resource
	ReadSeekCloser() (hugio.ReadSeekCloser, error)
}

// Resources represents a slice of resources, which can be a mix of different types.
// I.e. both pages and images etc.
type Resources []Resource

// ResourcesConverter converts a given slice of Resource objects to Resources.
type ResourcesConverter interface {
	ToResources() Resources
}

// ByType returns resources of a given resource type (ie. "image").
func (r Resources) ByType(tp string) Resources {
	var filtered Resources

	for _, resource := range r {
		if resource.ResourceType() == tp {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

// GetMatch finds the first Resource matching the given pattern, or nil if none found.
// See Match for a more complete explanation about the rules used.
func (r Resources) GetMatch(pattern string) Resource {
	g, err := getGlob(pattern)
	if err != nil {
		return nil
	}

	for _, resource := range r {
		if g.Match(strings.ToLower(resource.Name())) {
			return resource
		}
	}

	return nil
}

// Match gets all resources matching the given base filename prefix, e.g
// "*.png" will match all png files. The "*" does not match path delimiters (/),
// so if you organize your resources in sub-folders, you need to be explicit about it, e.g.:
// "images/*.png". To match any PNG image anywhere in the bundle you can do "**.png", and
// to match all PNG images below the images folder, use "images/**.jpg".
// The matching is case insensitive.
// Match matches by using the value of Resource.Name, which, by default, is a filename with
// path relative to the bundle root with Unix style slashes (/) and no leading slash, e.g. "images/logo.png".
// See https://github.com/gobwas/glob for the full rules set.
func (r Resources) Match(pattern string) Resources {
	g, err := getGlob(pattern)
	if err != nil {
		return nil
	}

	var matches Resources
	for _, resource := range r {
		if g.Match(strings.ToLower(resource.Name())) {
			matches = append(matches, resource)
		}
	}
	return matches
}

var (
	globCache = make(map[string]glob.Glob)
	globMu    sync.RWMutex
)

func getGlob(pattern string) (glob.Glob, error) {
	var g glob.Glob

	globMu.RLock()
	g, found := globCache[pattern]
	globMu.RUnlock()
	if !found {
		var err error
		g, err = glob.Compile(strings.ToLower(pattern), '/')
		if err != nil {
			return nil, err
		}

		globMu.Lock()
		globCache[pattern] = g
		globMu.Unlock()
	}

	return g, nil

}

// MergeByLanguage adds missing translations in r1 from r2.
func (r Resources) MergeByLanguage(r2 Resources) Resources {
	result := append(Resources(nil), r...)
	m := make(map[string]bool)
	for _, rr := range r {
		if translated, ok := rr.(translatedResource); ok {
			m[translated.TranslationKey()] = true
		}
	}

	for _, rr := range r2 {
		if translated, ok := rr.(translatedResource); ok {
			if _, found := m[translated.TranslationKey()]; !found {
				result = append(result, rr)
			}
		}
	}
	return result
}

// MergeByLanguageInterface is the generic version of MergeByLanguage. It
// is here just so it can be called from the tpl package.
func (r Resources) MergeByLanguageInterface(in interface{}) (interface{}, error) {
	r2, ok := in.(Resources)
	if !ok {
		return nil, fmt.Errorf("%T cannot be merged by language", in)
	}
	return r.MergeByLanguage(r2), nil
}

type Spec struct {
	*helpers.PathSpec

	MediaTypes    media.Types
	OutputFormats output.Formats

	Logger *loggers.Logger

	TextTemplates tpl.TemplateParseFinder

	// Holds default filter settings etc.
	imaging *Imaging

	imageCache    *imageCache
	ResourceCache *ResourceCache

	GenImagePath  string
	GenAssetsPath string
}

func NewSpec(s *helpers.PathSpec, logger *loggers.Logger, outputFormats output.Formats, mimeTypes media.Types) (*Spec, error) {

	imaging, err := decodeImaging(s.Cfg.GetStringMap("imaging"))
	if err != nil {
		return nil, err
	}

	if logger == nil {
		logger = loggers.NewErrorLogger()
	}

	genImagePath := filepath.FromSlash("_gen/images")
	// The transformed assets (CSS etc.)
	genAssetsPath := filepath.FromSlash("_gen/assets")

	rs := &Spec{PathSpec: s,
		Logger:        logger,
		GenImagePath:  genImagePath,
		GenAssetsPath: genAssetsPath,
		imaging:       &imaging,
		MediaTypes:    mimeTypes,
		OutputFormats: outputFormats,
		imageCache: newImageCache(
			s,
			// We're going to write a cache pruning routine later, so make it extremely
			// unlikely that the user shoots him or herself in the foot
			// and this is set to a value that represents data he/she
			// cares about. This should be set in stone once released.
			genImagePath,
		)}

	rs.ResourceCache = newResourceCache(rs)

	return rs, nil

}

type ResourceSourceDescriptor struct {
	// TargetPathBuilder is a callback to create target paths's relative to its owner.
	TargetPathBuilder func(base string) string

	// Need one of these to load the resource content.
	SourceFile         source.File
	OpenReadSeekCloser OpenReadSeekCloser

	// If OpenReadSeekerCloser is not set, we use this to open the file.
	SourceFilename string

	// The relative target filename without any language code.
	RelTargetFilename string

	// Any base path prepeneded to the permalink.
	// Typically the language code if this resource should be published to its sub-folder.
	URLBase string

	// Any base paths prepended to the target path. This will also typically be the
	// language code, but setting it here means that it should not have any effect on
	// the permalink.
	// This may be several values. In multihost mode we may publish the same resources to
	// multiple targets.
	TargetBasePaths []string

	// Delay publishing until either Permalink or RelPermalink is called. Maybe never.
	LazyPublish bool
}

func (r ResourceSourceDescriptor) Filename() string {
	if r.SourceFile != nil {
		return r.SourceFile.Filename()
	}
	return r.SourceFilename
}

func (r *Spec) sourceFs() afero.Fs {
	return r.PathSpec.BaseFs.Content.Fs
}

func (r *Spec) New(fd ResourceSourceDescriptor) (Resource, error) {
	return r.newResourceForFs(r.sourceFs(), fd)
}

func (r *Spec) NewForFs(sourceFs afero.Fs, fd ResourceSourceDescriptor) (Resource, error) {
	return r.newResourceForFs(sourceFs, fd)
}

func (r *Spec) newResourceForFs(sourceFs afero.Fs, fd ResourceSourceDescriptor) (Resource, error) {
	if fd.OpenReadSeekCloser == nil {
		if fd.SourceFile != nil && fd.SourceFilename != "" {
			return nil, errors.New("both SourceFile and AbsSourceFilename provided")
		} else if fd.SourceFile == nil && fd.SourceFilename == "" {
			return nil, errors.New("either SourceFile or AbsSourceFilename must be provided")
		}
	}

	if fd.RelTargetFilename == "" {
		fd.RelTargetFilename = fd.Filename()
	}

	if len(fd.TargetBasePaths) == 0 {
		// If not set, we publish the same resource to all hosts.
		fd.TargetBasePaths = r.MultihostTargetBasePaths
	}

	return r.newResource(sourceFs, fd)
}

func (r *Spec) newResource(sourceFs afero.Fs, fd ResourceSourceDescriptor) (Resource, error) {
	var fi os.FileInfo
	var sourceFilename string

	if fd.OpenReadSeekCloser != nil {

	} else if fd.SourceFilename != "" {
		var err error
		fi, err = sourceFs.Stat(fd.SourceFilename)
		if err != nil {
			return nil, err
		}
		sourceFilename = fd.SourceFilename
	} else {
		fi = fd.SourceFile.FileInfo()
		sourceFilename = fd.SourceFile.Filename()
	}

	if fd.RelTargetFilename == "" {
		fd.RelTargetFilename = sourceFilename
	}

	ext := filepath.Ext(fd.RelTargetFilename)
	mimeType, found := r.MediaTypes.GetFirstBySuffix(strings.TrimPrefix(ext, "."))
	// TODO(bep) we need to handle these ambigous types better, but in this context
	// we most likely want the application/xml type.
	if mimeType.Suffix() == "xml" && mimeType.SubType == "rss" {
		mimeType, found = r.MediaTypes.GetByType("application/xml")
	}

	if !found {
		mimeStr := mime.TypeByExtension(ext)
		if mimeStr != "" {
			mimeType, _ = media.FromStringAndExt(mimeStr, ext)
		}
	}

	gr := r.newGenericResourceWithBase(
		sourceFs,
		fd.LazyPublish,
		fd.OpenReadSeekCloser,
		fd.URLBase,
		fd.TargetBasePaths,
		fd.TargetPathBuilder,
		fi,
		sourceFilename,
		fd.RelTargetFilename,
		mimeType)

	if mimeType.MainType == "image" {
		ext := strings.ToLower(helpers.Ext(sourceFilename))

		imgFormat, ok := imageFormats[ext]
		if !ok {
			// This allows SVG etc. to be used as resources. They will not have the methods of the Image, but
			// that would not (currently) have worked.
			return gr, nil
		}

		if err := gr.initHash(); err != nil {
			return nil, err
		}

		return &Image{
			format:          imgFormat,
			imaging:         r.imaging,
			genericResource: gr}, nil
	}
	return gr, nil

}

// TODO(bep) unify
func (r *Spec) IsInImageCache(key string) bool {
	// This is used for cache pruning. We currently only have images, but we could
	// imagine expanding on this.
	return r.imageCache.isInCache(key)
}

func (r *Spec) DeleteCacheByPrefix(prefix string) {
	r.imageCache.deleteByPrefix(prefix)
}

func (r *Spec) ClearCaches() {
	r.imageCache.clear()
	r.ResourceCache.clear()
}

func (r *Spec) CacheStats() string {
	r.imageCache.mu.RLock()
	defer r.imageCache.mu.RUnlock()

	s := fmt.Sprintf("Cache entries: %d", len(r.imageCache.store))

	count := 0
	for k := range r.imageCache.store {
		if count > 5 {
			break
		}
		s += "\n" + k
		count++
	}

	return s
}

type dirFile struct {
	// This is the directory component with Unix-style slashes.
	dir string
	// This is the file component.
	file string
}

func (d dirFile) path() string {
	return path.Join(d.dir, d.file)
}

type resourcePathDescriptor struct {
	// The relative target directory and filename.
	relTargetDirFile dirFile

	// Callback used to construct a target path relative to its owner.
	targetPathBuilder func(rel string) string

	// baseURLDir is the fixed sub-folder for a resource in permalinks. This will typically
	// be the language code if we publish to the language's sub-folder.
	baseURLDir string

	// This will normally be the same as above, but this will only apply to publishing
	// of resources. It may be mulltiple values when in multihost mode.
	baseTargetPathDirs []string

	// baseOffset is set when the output format's path has a offset, e.g. for AMP.
	baseOffset string
}

type resourceContent struct {
	content     string
	contentInit sync.Once
}

type resourceHash struct {
	hash     string
	hashInit sync.Once
}

type publishOnce struct {
	publisherInit sync.Once
	publisherErr  error
	logger        *loggers.Logger
}

func (l *publishOnce) publish(s Source) error {
	l.publisherInit.Do(func() {
		l.publisherErr = s.Publish()
		if l.publisherErr != nil {
			l.logger.ERROR.Printf("failed to publish Resource: %s", l.publisherErr)
		}
	})
	return l.publisherErr
}

// genericResource represents a generic linkable resource.
type genericResource struct {
	commonResource
	resourcePathDescriptor

	title  string
	name   string
	params map[string]interface{}

	// Absolute filename to the source, including any content folder path.
	// Note that this is absolute in relation to the filesystem it is stored in.
	// It can be a base path filesystem, and then this filename will not match
	// the path to the file on the real filesystem.
	sourceFilename string

	// Will be set if this resource is backed by something other than a file.
	openReadSeekerCloser OpenReadSeekCloser

	// A hash of the source content. Is only calculated in caching situations.
	*resourceHash

	// This may be set to tell us to look in another filesystem for this resource.
	// We, by default, use the sourceFs filesystem in the spec below.
	overriddenSourceFs afero.Fs

	spec *Spec

	resourceType string
	mediaType    media.Type

	osFileInfo os.FileInfo

	// We create copies of this struct, so this needs to be a pointer.
	*resourceContent

	// May be set to signal lazy/delayed publishing.
	*publishOnce
}

type commonResource struct {
}

func (l *genericResource) Data() interface{} {
	return noData
}

func (l *genericResource) Content() (interface{}, error) {
	if err := l.initContent(); err != nil {
		return nil, err
	}

	return l.content, nil
}

func (l *genericResource) ReadSeekCloser() (hugio.ReadSeekCloser, error) {
	if l.openReadSeekerCloser != nil {
		return l.openReadSeekerCloser()
	}
	f, err := l.sourceFs().Open(l.sourceFilename)
	if err != nil {
		return nil, err
	}
	return f, nil

}

func (l *genericResource) MediaType() media.Type {
	return l.mediaType
}

// Implement the Cloner interface.
func (l genericResource) WithNewBase(base string) Resource {
	l.baseOffset = base
	l.resourceContent = &resourceContent{}
	return &l
}

// Slice is not meant to be used externally. It's a bridge function
// for the template functions. See collections.Slice.
func (commonResource) Slice(in interface{}) (interface{}, error) {
	switch items := in.(type) {
	case Resources:
		return items, nil
	case []interface{}:
		groups := make(Resources, len(items))
		for i, v := range items {
			g, ok := v.(Resource)
			if !ok {
				return nil, fmt.Errorf("type %T is not a Resource", v)
			}
			groups[i] = g
		}
		return groups, nil
	default:
		return nil, fmt.Errorf("invalid slice type %T", items)
	}
}

func (l *genericResource) initHash() error {
	var err error
	l.hashInit.Do(func() {
		var hash string
		var f hugio.ReadSeekCloser
		f, err = l.ReadSeekCloser()
		if err != nil {
			err = errors.Wrap(err, "failed to open source file")
			return
		}
		defer f.Close()

		hash, err = helpers.MD5FromFileFast(f)
		if err != nil {
			return
		}
		l.hash = hash

	})

	return err
}

func (l *genericResource) initContent() error {
	var err error
	l.contentInit.Do(func() {
		var r hugio.ReadSeekCloser
		r, err = l.ReadSeekCloser()
		if err != nil {
			return
		}
		defer r.Close()

		var b []byte
		b, err = ioutil.ReadAll(r)
		if err != nil {
			return
		}

		l.content = string(b)

	})

	return err
}

func (l *genericResource) sourceFs() afero.Fs {
	if l.overriddenSourceFs != nil {
		return l.overriddenSourceFs
	}
	return l.spec.sourceFs()
}

func (l *genericResource) publishIfNeeded() {
	if l.publishOnce != nil {
		l.publishOnce.publish(l)
	}
}

func (l *genericResource) Permalink() string {
	l.publishIfNeeded()
	return l.spec.PermalinkForBaseURL(l.relPermalinkForRel(l.relTargetDirFile.path()), l.spec.BaseURL.HostURL())
}

func (l *genericResource) RelPermalink() string {
	l.publishIfNeeded()
	return l.relPermalinkFor(l.relTargetDirFile.path())
}

func (l *genericResource) relPermalinkFor(target string) string {
	return l.relPermalinkForRel(target)

}
func (l *genericResource) permalinkFor(target string) string {
	return l.spec.PermalinkForBaseURL(l.relPermalinkForRel(target), l.spec.BaseURL.HostURL())

}
func (l *genericResource) relTargetPathsFor(target string) []string {
	return l.relTargetPathsForRel(target)
}

func (l *genericResource) relTargetPaths() []string {
	return l.relTargetPathsForRel(l.targetPath())
}

func (l *genericResource) Name() string {
	return l.name
}

func (l *genericResource) Title() string {
	return l.title
}

func (l *genericResource) Params() map[string]interface{} {
	return l.params
}

func (l *genericResource) setTitle(title string) {
	l.title = title
}

func (l *genericResource) setName(name string) {
	l.name = name
}

func (l *genericResource) updateParams(params map[string]interface{}) {
	if l.params == nil {
		l.params = params
		return
	}

	// Sets the params not already set
	for k, v := range params {
		if _, found := l.params[k]; !found {
			l.params[k] = v
		}
	}
}

func (l *genericResource) relPermalinkForRel(rel string) string {
	return l.spec.PathSpec.URLizeFilename(l.relTargetPathForRel(rel, false, true))
}

func (l *genericResource) relTargetPathsForRel(rel string) []string {
	if len(l.baseTargetPathDirs) == 0 {
		return []string{l.relTargetPathForRelAndBasePath(rel, "", false)}
	}

	var targetPaths = make([]string, len(l.baseTargetPathDirs))
	for i, dir := range l.baseTargetPathDirs {
		targetPaths[i] = l.relTargetPathForRelAndBasePath(rel, dir, false)
	}
	return targetPaths
}

func (l *genericResource) relTargetPathForRel(rel string, addBaseTargetPath, isURL bool) string {
	if addBaseTargetPath && len(l.baseTargetPathDirs) > 1 {
		panic("multiple baseTargetPathDirs")
	}
	var basePath string
	if addBaseTargetPath && len(l.baseTargetPathDirs) > 0 {
		basePath = l.baseTargetPathDirs[0]
	}

	return l.relTargetPathForRelAndBasePath(rel, basePath, isURL)
}

func (l *genericResource) relTargetPathForRelAndBasePath(rel, basePath string, isURL bool) string {
	if l.targetPathBuilder != nil {
		rel = l.targetPathBuilder(rel)
	}

	if isURL && l.baseURLDir != "" {
		rel = path.Join(l.baseURLDir, rel)
	}

	if basePath != "" {
		rel = path.Join(basePath, rel)
	}

	if l.baseOffset != "" {
		rel = path.Join(l.baseOffset, rel)
	}

	if isURL && l.spec.PathSpec.BasePath != "" {
		rel = path.Join(l.spec.PathSpec.BasePath, rel)
	}

	if len(rel) == 0 || rel[0] != '/' {
		rel = "/" + rel
	}

	return rel
}

func (l *genericResource) ResourceType() string {
	return l.resourceType
}

func (l *genericResource) String() string {
	return fmt.Sprintf("Resource(%s: %s)", l.resourceType, l.name)
}

func (l *genericResource) Publish() error {
	fr, err := l.ReadSeekCloser()
	if err != nil {
		return err
	}
	defer fr.Close()
	fw, err := helpers.OpenFilesForWriting(l.spec.BaseFs.PublishFs, l.targetFilenames()...)
	if err != nil {
		return err
	}
	defer fw.Close()

	_, err = io.Copy(fw, fr)
	return err
}

// Path is stored with Unix style slashes.
func (l *genericResource) targetPath() string {
	return l.relTargetDirFile.path()
}

func (l *genericResource) targetFilenames() []string {
	paths := l.relTargetPaths()
	for i, p := range paths {
		paths[i] = filepath.Clean(p)
	}
	return paths
}

// TODO(bep) clean up below
func (r *Spec) newGenericResource(sourceFs afero.Fs,
	targetPathBuilder func(base string) string,
	osFileInfo os.FileInfo,
	sourceFilename,
	baseFilename string,
	mediaType media.Type) *genericResource {
	return r.newGenericResourceWithBase(
		sourceFs,
		false,
		nil,
		"",
		nil,
		targetPathBuilder,
		osFileInfo,
		sourceFilename,
		baseFilename,
		mediaType,
	)

}

func (r *Spec) newGenericResourceWithBase(
	sourceFs afero.Fs,
	lazyPublish bool,
	openReadSeekerCloser OpenReadSeekCloser,
	urlBaseDir string,
	targetPathBaseDirs []string,
	targetPathBuilder func(base string) string,
	osFileInfo os.FileInfo,
	sourceFilename,
	baseFilename string,
	mediaType media.Type) *genericResource {

	// This value is used both to construct URLs and file paths, but start
	// with a Unix-styled path.
	baseFilename = helpers.ToSlashTrimLeading(baseFilename)
	fpath, fname := path.Split(baseFilename)

	var resourceType string
	if mediaType.MainType == "image" {
		resourceType = mediaType.MainType
	} else {
		resourceType = mediaType.SubType
	}

	pathDescriptor := resourcePathDescriptor{
		baseURLDir:         urlBaseDir,
		baseTargetPathDirs: targetPathBaseDirs,
		targetPathBuilder:  targetPathBuilder,
		relTargetDirFile:   dirFile{dir: fpath, file: fname},
	}

	var po *publishOnce
	if lazyPublish {
		po = &publishOnce{logger: r.Logger}
	}

	return &genericResource{
		openReadSeekerCloser:   openReadSeekerCloser,
		publishOnce:            po,
		resourcePathDescriptor: pathDescriptor,
		overriddenSourceFs:     sourceFs,
		osFileInfo:             osFileInfo,
		sourceFilename:         sourceFilename,
		mediaType:              mediaType,
		resourceType:           resourceType,
		spec:                   r,
		params:                 make(map[string]interface{}),
		name:                   baseFilename,
		title:                  baseFilename,
		resourceContent:        &resourceContent{},
		resourceHash:           &resourceHash{},
	}
}

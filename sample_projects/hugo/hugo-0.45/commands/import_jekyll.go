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

package commands

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/hugofs"
	"github.com/gohugoio/hugo/hugolib"
	"github.com/gohugoio/hugo/parser"
	"github.com/spf13/afero"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var _ cmder = (*importCmd)(nil)

type importCmd struct {
	*baseCmd
}

func newImportCmd() *importCmd {
	cc := &importCmd{}

	cc.baseCmd = newBaseCmd(&cobra.Command{
		Use:   "import",
		Short: "Import your site from others.",
		Long: `Import your site from other web site generators like Jekyll.

Import requires a subcommand, e.g. ` + "`hugo import jekyll jekyll_root_path target_path`.",
		RunE: nil,
	})

	importJekyllCmd := &cobra.Command{
		Use:   "jekyll",
		Short: "hugo import from Jekyll",
		Long: `hugo import from Jekyll.

Import from Jekyll requires two paths, e.g. ` + "`hugo import jekyll jekyll_root_path target_path`.",
		RunE: cc.importFromJekyll,
	}

	importJekyllCmd.Flags().Bool("force", false, "allow import into non-empty target directory")

	cc.cmd.AddCommand(importJekyllCmd)

	return cc

}

func (i *importCmd) importFromJekyll(cmd *cobra.Command, args []string) error {

	if len(args) < 2 {
		return newUserError(`Import from Jekyll requires two paths, e.g. ` + "`hugo import jekyll jekyll_root_path target_path`.")
	}

	jekyllRoot, err := filepath.Abs(filepath.Clean(args[0]))
	if err != nil {
		return newUserError("Path error:", args[0])
	}

	targetDir, err := filepath.Abs(filepath.Clean(args[1]))
	if err != nil {
		return newUserError("Path error:", args[1])
	}

	jww.INFO.Println("Import Jekyll from:", jekyllRoot, "to:", targetDir)

	if strings.HasPrefix(filepath.Dir(targetDir), jekyllRoot) {
		return newUserError("Target path should not be inside the Jekyll root, aborting.")
	}

	forceImport, _ := cmd.Flags().GetBool("force")

	fs := afero.NewOsFs()
	jekyllPostDirs, hasAnyPost := i.getJekyllDirInfo(fs, jekyllRoot)
	if !hasAnyPost {
		return errors.New("Your Jekyll root contains neither posts nor drafts, aborting.")
	}

	site, err := i.createSiteFromJekyll(jekyllRoot, targetDir, jekyllPostDirs, forceImport)

	if err != nil {
		return newUserError(err)
	}

	jww.FEEDBACK.Println("Importing...")

	fileCount := 0
	callback := func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(jekyllRoot, path)
		if err != nil {
			return newUserError("Get rel path error:", path)
		}

		relPath = filepath.ToSlash(relPath)
		draft := false

		switch {
		case strings.Contains(relPath, "_posts/"):
			relPath = filepath.Join("content/post", strings.Replace(relPath, "_posts/", "", -1))
		case strings.Contains(relPath, "_drafts/"):
			relPath = filepath.Join("content/draft", strings.Replace(relPath, "_drafts/", "", -1))
			draft = true
		default:
			return nil
		}

		fileCount++
		return convertJekyllPost(site, path, relPath, targetDir, draft)
	}

	for jekyllPostDir, hasAnyPostInDir := range jekyllPostDirs {
		if hasAnyPostInDir {
			if err = helpers.SymbolicWalk(hugofs.Os, filepath.Join(jekyllRoot, jekyllPostDir), callback); err != nil {
				return err
			}
		}
	}

	jww.FEEDBACK.Println("Congratulations!", fileCount, "post(s) imported!")
	jww.FEEDBACK.Println("Now, start Hugo by yourself:\n" +
		"$ git clone https://github.com/spf13/herring-cove.git " + args[1] + "/themes/herring-cove")
	jww.FEEDBACK.Println("$ cd " + args[1] + "\n$ hugo server --theme=herring-cove")

	return nil
}

func (i *importCmd) getJekyllDirInfo(fs afero.Fs, jekyllRoot string) (map[string]bool, bool) {
	postDirs := make(map[string]bool)
	hasAnyPost := false
	if entries, err := ioutil.ReadDir(jekyllRoot); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				subDir := filepath.Join(jekyllRoot, entry.Name())
				if isPostDir, hasAnyPostInDir := i.retrieveJekyllPostDir(fs, subDir); isPostDir {
					postDirs[entry.Name()] = hasAnyPostInDir
					if hasAnyPostInDir {
						hasAnyPost = true
					}
				}
			}
		}
	}
	return postDirs, hasAnyPost
}

func (i *importCmd) retrieveJekyllPostDir(fs afero.Fs, dir string) (bool, bool) {
	if strings.HasSuffix(dir, "_posts") || strings.HasSuffix(dir, "_drafts") {
		isEmpty, _ := helpers.IsEmpty(dir, fs)
		return true, !isEmpty
	}

	if entries, err := ioutil.ReadDir(dir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				subDir := filepath.Join(dir, entry.Name())
				if isPostDir, hasAnyPost := i.retrieveJekyllPostDir(fs, subDir); isPostDir {
					return isPostDir, hasAnyPost
				}
			}
		}
	}

	return false, true
}

func (i *importCmd) createSiteFromJekyll(jekyllRoot, targetDir string, jekyllPostDirs map[string]bool, force bool) (*hugolib.Site, error) {
	s, err := hugolib.NewSiteDefaultLang()
	if err != nil {
		return nil, err
	}

	fs := s.Fs.Source
	if exists, _ := helpers.Exists(targetDir, fs); exists {
		if isDir, _ := helpers.IsDir(targetDir, fs); !isDir {
			return nil, errors.New("Target path \"" + targetDir + "\" already exists but not a directory")
		}

		isEmpty, _ := helpers.IsEmpty(targetDir, fs)

		if !isEmpty && !force {
			return nil, errors.New("Target path \"" + targetDir + "\" already exists and is not empty")
		}
	}

	jekyllConfig := i.loadJekyllConfig(fs, jekyllRoot)

	mkdir(targetDir, "layouts")
	mkdir(targetDir, "content")
	mkdir(targetDir, "archetypes")
	mkdir(targetDir, "static")
	mkdir(targetDir, "data")
	mkdir(targetDir, "themes")

	i.createConfigFromJekyll(fs, targetDir, "yaml", jekyllConfig)

	i.copyJekyllFilesAndFolders(jekyllRoot, filepath.Join(targetDir, "static"), jekyllPostDirs)

	return s, nil
}

func (i *importCmd) loadJekyllConfig(fs afero.Fs, jekyllRoot string) map[string]interface{} {
	path := filepath.Join(jekyllRoot, "_config.yml")

	exists, err := helpers.Exists(path, fs)

	if err != nil || !exists {
		jww.WARN.Println("_config.yaml not found: Is the specified Jekyll root correct?")
		return nil
	}

	f, err := fs.Open(path)
	if err != nil {
		return nil
	}

	defer f.Close()

	b, err := ioutil.ReadAll(f)

	if err != nil {
		return nil
	}

	c, err := parser.HandleYAMLMetaData(b)

	if err != nil {
		return nil
	}

	return c
}

func (i *importCmd) createConfigFromJekyll(fs afero.Fs, inpath string, kind string, jekyllConfig map[string]interface{}) (err error) {
	title := "My New Hugo Site"
	baseURL := "http://example.org/"

	for key, value := range jekyllConfig {
		lowerKey := strings.ToLower(key)

		switch lowerKey {
		case "title":
			if str, ok := value.(string); ok {
				title = str
			}

		case "url":
			if str, ok := value.(string); ok {
				baseURL = str
			}
		}
	}

	in := map[string]interface{}{
		"baseURL":            baseURL,
		"title":              title,
		"languageCode":       "en-us",
		"disablePathToLower": true,
	}
	kind = parser.FormatSanitize(kind)

	var buf bytes.Buffer
	err = parser.InterfaceToConfig(in, parser.FormatToLeadRune(kind), &buf)
	if err != nil {
		return err
	}

	return helpers.WriteToDisk(filepath.Join(inpath, "config."+kind), &buf, fs)
}

func copyFile(source string, dest string) error {
	sf, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sf.Close()
	df, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer df.Close()
	_, err = io.Copy(df, sf)
	if err == nil {
		si, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, si.Mode())

			if err != nil {
				return err
			}
		}

	}
	return nil
}

func copyDir(source string, dest string) error {
	fi, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return errors.New(source + " is not a directory")
	}
	err = os.MkdirAll(dest, fi.Mode())
	if err != nil {
		return err
	}
	entries, err := ioutil.ReadDir(source)
	for _, entry := range entries {
		sfp := filepath.Join(source, entry.Name())
		dfp := filepath.Join(dest, entry.Name())
		if entry.IsDir() {
			err = copyDir(sfp, dfp)
			if err != nil {
				jww.ERROR.Println(err)
			}
		} else {
			err = copyFile(sfp, dfp)
			if err != nil {
				jww.ERROR.Println(err)
			}
		}

	}
	return nil
}

func (i *importCmd) copyJekyllFilesAndFolders(jekyllRoot, dest string, jekyllPostDirs map[string]bool) (err error) {
	fi, err := os.Stat(jekyllRoot)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return errors.New(jekyllRoot + " is not a directory")
	}
	err = os.MkdirAll(dest, fi.Mode())
	if err != nil {
		return err
	}
	entries, err := ioutil.ReadDir(jekyllRoot)
	for _, entry := range entries {
		sfp := filepath.Join(jekyllRoot, entry.Name())
		dfp := filepath.Join(dest, entry.Name())
		if entry.IsDir() {
			if entry.Name()[0] != '_' && entry.Name()[0] != '.' {
				if _, ok := jekyllPostDirs[entry.Name()]; !ok {
					err = copyDir(sfp, dfp)
					if err != nil {
						jww.ERROR.Println(err)
					}
				}
			}
		} else {
			lowerEntryName := strings.ToLower(entry.Name())
			exceptSuffix := []string{".md", ".markdown", ".html", ".htm",
				".xml", ".textile", "rakefile", "gemfile", ".lock"}
			isExcept := false
			for _, suffix := range exceptSuffix {
				if strings.HasSuffix(lowerEntryName, suffix) {
					isExcept = true
					break
				}
			}

			if !isExcept && entry.Name()[0] != '.' && entry.Name()[0] != '_' {
				err = copyFile(sfp, dfp)
				if err != nil {
					jww.ERROR.Println(err)
				}
			}
		}

	}
	return nil
}

func parseJekyllFilename(filename string) (time.Time, string, error) {
	re := regexp.MustCompile(`(\d+-\d+-\d+)-(.+)\..*`)
	r := re.FindAllStringSubmatch(filename, -1)
	if len(r) == 0 {
		return time.Now(), "", errors.New("filename not match")
	}

	postDate, err := time.Parse("2006-1-2", r[0][1])
	if err != nil {
		return time.Now(), "", err
	}

	postName := r[0][2]

	return postDate, postName, nil
}

func convertJekyllPost(s *hugolib.Site, path, relPath, targetDir string, draft bool) error {
	jww.TRACE.Println("Converting", path)

	filename := filepath.Base(path)
	postDate, postName, err := parseJekyllFilename(filename)
	if err != nil {
		jww.WARN.Printf("Failed to parse filename '%s': %s. Skipping.", filename, err)
		return nil
	}

	jww.TRACE.Println(filename, postDate, postName)

	targetFile := filepath.Join(targetDir, relPath)
	targetParentDir := filepath.Dir(targetFile)
	os.MkdirAll(targetParentDir, 0777)

	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		jww.ERROR.Println("Read file error:", path)
		return err
	}

	psr, err := parser.ReadFrom(bytes.NewReader(contentBytes))
	if err != nil {
		jww.ERROR.Println("Parse file error:", path)
		return err
	}

	metadata, err := psr.Metadata()
	if err != nil {
		jww.ERROR.Println("Processing file error:", path)
		return err
	}

	newmetadata, err := convertJekyllMetaData(metadata, postName, postDate, draft)
	if err != nil {
		jww.ERROR.Println("Convert metadata error:", path)
		return err
	}

	jww.TRACE.Println(newmetadata)
	content := convertJekyllContent(newmetadata, string(psr.Content()))

	page, err := s.NewPage(filename)
	if err != nil {
		jww.ERROR.Println("New page error", filename)
		return err
	}

	page.SetSourceContent([]byte(content))
	page.SetSourceMetaData(newmetadata, parser.FormatToLeadRune("yaml"))
	page.SaveSourceAs(targetFile)

	jww.TRACE.Println("Target file:", targetFile)

	return nil
}

func convertJekyllMetaData(m interface{}, postName string, postDate time.Time, draft bool) (interface{}, error) {
	metadata, err := cast.ToStringMapE(m)
	if err != nil {
		return nil, err
	}

	if draft {
		metadata["draft"] = true
	}

	for key, value := range metadata {
		lowerKey := strings.ToLower(key)

		switch lowerKey {
		case "layout":
			delete(metadata, key)
		case "permalink":
			if str, ok := value.(string); ok {
				metadata["url"] = str
			}
			delete(metadata, key)
		case "category":
			if str, ok := value.(string); ok {
				metadata["categories"] = []string{str}
			}
			delete(metadata, key)
		case "excerpt_separator":
			if key != lowerKey {
				delete(metadata, key)
				metadata[lowerKey] = value
			}
		case "date":
			if str, ok := value.(string); ok {
				re := regexp.MustCompile(`(\d+):(\d+):(\d+)`)
				r := re.FindAllStringSubmatch(str, -1)
				if len(r) > 0 {
					hour, _ := strconv.Atoi(r[0][1])
					minute, _ := strconv.Atoi(r[0][2])
					second, _ := strconv.Atoi(r[0][3])
					postDate = time.Date(postDate.Year(), postDate.Month(), postDate.Day(), hour, minute, second, 0, time.UTC)
				}
			}
			delete(metadata, key)
		}

	}

	metadata["date"] = postDate.Format(time.RFC3339)

	return metadata, nil
}

func convertJekyllContent(m interface{}, content string) string {
	metadata, _ := cast.ToStringMapE(m)

	lines := strings.Split(content, "\n")
	var resultLines []string
	for _, line := range lines {
		resultLines = append(resultLines, strings.Trim(line, "\r\n"))
	}

	content = strings.Join(resultLines, "\n")

	excerptSep := "<!--more-->"
	if value, ok := metadata["excerpt_separator"]; ok {
		if str, strOk := value.(string); strOk {
			content = strings.Replace(content, strings.TrimSpace(str), excerptSep, -1)
		}
	}

	replaceList := []struct {
		re      *regexp.Regexp
		replace string
	}{
		{regexp.MustCompile("(?i)<!-- more -->"), "<!--more-->"},
		{regexp.MustCompile(`\{%\s*raw\s*%\}\s*(.*?)\s*\{%\s*endraw\s*%\}`), "$1"},
		{regexp.MustCompile(`{%\s*highlight\s*(.*?)\s*%}`), "{{< highlight $1 >}}"},
		{regexp.MustCompile(`{%\s*endhighlight\s*%}`), "{{< / highlight >}}"},
	}

	for _, replace := range replaceList {
		content = replace.re.ReplaceAllString(content, replace.replace)
	}

	replaceListFunc := []struct {
		re      *regexp.Regexp
		replace func(string) string
	}{
		// Octopress image tag: http://octopress.org/docs/plugins/image-tag/
		{regexp.MustCompile(`{%\s+img\s*(.*?)\s*%}`), replaceImageTag},
	}

	for _, replace := range replaceListFunc {
		content = replace.re.ReplaceAllStringFunc(content, replace.replace)
	}

	return content
}

func replaceImageTag(match string) string {
	r := regexp.MustCompile(`{%\s+img\s*(\p{L}*)\s+([\S]*/[\S]+)\s+(\d*)\s*(\d*)\s*(.*?)\s*%}`)
	result := bytes.NewBufferString("{{< figure ")
	parts := r.FindStringSubmatch(match)
	// Index 0 is the entire string, ignore
	replaceOptionalPart(result, "class", parts[1])
	replaceOptionalPart(result, "src", parts[2])
	replaceOptionalPart(result, "width", parts[3])
	replaceOptionalPart(result, "height", parts[4])
	// title + alt
	part := parts[5]
	if len(part) > 0 {
		splits := strings.Split(part, "'")
		lenSplits := len(splits)
		if lenSplits == 1 {
			replaceOptionalPart(result, "title", splits[0])
		} else if lenSplits == 3 {
			replaceOptionalPart(result, "title", splits[1])
		} else if lenSplits == 5 {
			replaceOptionalPart(result, "title", splits[1])
			replaceOptionalPart(result, "alt", splits[3])
		}
	}
	result.WriteString(">}}")
	return result.String()

}
func replaceOptionalPart(buffer *bytes.Buffer, partName string, part string) {
	if len(part) > 0 {
		buffer.WriteString(partName + "=\"" + part + "\" ")
	}
}

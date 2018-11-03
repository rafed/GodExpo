package hugolib

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestMultihosts(t *testing.T) {
	t.Parallel()

	var multiSiteTOMLConfigTemplate = `
paginate = 1
disablePathToLower = true
defaultContentLanguage = "{{ .DefaultContentLanguage }}"
defaultContentLanguageInSubdir = {{ .DefaultContentLanguageInSubdir }}
staticDir = ["s1", "s2"]

[permalinks]
other = "/somewhere/else/:filename"

[Taxonomies]
tag = "tags"

[Languages]
[Languages.en]
staticDir2 = ["ens1", "ens2"]
baseURL = "https://example.com/docs"
weight = 10
title = "In English"
languageName = "English"

[Languages.fr]
staticDir2 = ["frs1", "frs2"]
baseURL = "https://example.fr"
weight = 20
title = "Le Français"
languageName = "Français"

[Languages.nn]
staticDir2 = ["nns1", "nns2"]
baseURL = "https://example.no"
weight = 30
title = "På nynorsk"
languageName = "Nynorsk"

`

	siteConfig := testSiteConfig{Running: true, Fs: afero.NewMemMapFs(), DefaultContentLanguage: "fr", DefaultContentLanguageInSubdir: false}
	sites := createMultiTestSites(t, siteConfig, multiSiteTOMLConfigTemplate)
	fs := sites.Fs
	th := testHelper{sites.Cfg, fs, t}
	assert := require.New(t)
	cfg := BuildCfg{}
	err := sites.Build(cfg)
	assert.NoError(err)

	th.assertFileContent("public/en/sect/doc1-slug/index.html", "Hello")

	s1 := sites.Sites[0]

	assert.Equal([]string{"s1", "s2", "ens1", "ens2"}, s1.StaticDirs())

	s1h := s1.getPage(KindHome)
	assert.True(s1h.IsTranslated())
	assert.Len(s1h.Translations(), 2)
	assert.Equal("https://example.com/docs/", s1h.Permalink())

	// For “regular multilingual” we kept the aliases pages with url in front matter
	// as a literal value that we use as is.
	// There is an ambiguity in the guessing.
	// For multihost, we never want any content in the root.
	//
	// check url in front matter:
	pageWithURLInFrontMatter := s1.getPage(KindPage, "sect/doc3.en.md")
	assert.NotNil(pageWithURLInFrontMatter)
	assert.Equal("/superbob", pageWithURLInFrontMatter.URL())
	assert.Equal("/docs/superbob/", pageWithURLInFrontMatter.RelPermalink())
	th.assertFileContent("public/en/superbob/index.html", "doc3|Hello|en")

	// check alias:
	th.assertFileContent("public/en/al/alias1/index.html", `content="0; url=https://example.com/docs/superbob/"`)
	th.assertFileContent("public/en/al/alias2/index.html", `content="0; url=https://example.com/docs/superbob/"`)

	s2 := sites.Sites[1]
	assert.Equal([]string{"s1", "s2", "frs1", "frs2"}, s2.StaticDirs())

	s2h := s2.getPage(KindHome)
	assert.Equal("https://example.fr/", s2h.Permalink())

	th.assertFileContentStraight("public/fr/index.html", "French Home Page")
	th.assertFileContentStraight("public/en/index.html", "Default Home Page")

	// Check paginators
	th.assertFileContent("public/en/page/1/index.html", `refresh" content="0; url=https://example.com/docs/"`)
	th.assertFileContent("public/nn/page/1/index.html", `refresh" content="0; url=https://example.no/"`)
	th.assertFileContent("public/en/sect/page/2/index.html", "List Page 2", "Hello", "https://example.com/docs/sect/", "\"/docs/sect/page/3/")
	th.assertFileContent("public/fr/sect/page/2/index.html", "List Page 2", "Bonjour", "https://example.fr/sect/")

	// Check bundles

	bundleEn := s1.getPage(KindPage, "bundles/b1/index.en.md")
	require.NotNil(t, bundleEn)
	require.Equal(t, "/docs/bundles/b1/", bundleEn.RelPermalink())
	require.Equal(t, 1, len(bundleEn.Resources))
	logoEn := bundleEn.Resources.GetByPrefix("logo")
	require.NotNil(t, logoEn)
	require.Equal(t, "/docs/bundles/b1/logo.png", logoEn.RelPermalink())
	require.Contains(t, readFileFromFs(t, fs.Destination, filepath.FromSlash("public/en/bundles/b1/logo.png")), "PNG Data")

	bundleFr := s2.getPage(KindPage, "bundles/b1/index.md")
	require.NotNil(t, bundleFr)
	require.Equal(t, "/bundles/b1/", bundleFr.RelPermalink())
	require.Equal(t, 1, len(bundleFr.Resources))
	logoFr := bundleFr.Resources.GetByPrefix("logo")
	require.NotNil(t, logoFr)
	require.Equal(t, "/bundles/b1/logo.png", logoFr.RelPermalink())
	require.Contains(t, readFileFromFs(t, fs.Destination, filepath.FromSlash("public/fr/bundles/b1/logo.png")), "PNG Data")

}

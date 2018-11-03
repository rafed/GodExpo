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

// Package releaser implements a set of utilities and a wrapper around Goreleaser
// to help automate the Hugo release process.
package releaser

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gohugoio/hugo/helpers"
)

const commitPrefix = "releaser:"

type releaseNotesState int

const (
	releaseNotesNone = iota
	releaseNotesCreated
	releaseNotesReady
)

// ReleaseHandler provides functionality to release a new version of Hugo.
type ReleaseHandler struct {
	cliVersion string

	skipPublish bool

	// Just simulate, no actual changes.
	try bool

	git func(args ...string) (string, error)
}

func (r ReleaseHandler) calculateVersions() (helpers.HugoVersion, helpers.HugoVersion) {
	newVersion := helpers.MustParseHugoVersion(r.cliVersion)
	finalVersion := newVersion.Next()
	finalVersion.PatchLevel = 0

	if newVersion.Suffix != "-test" {
		newVersion.Suffix = ""
	}

	finalVersion.Suffix = "-DEV"

	return newVersion, finalVersion
}

// New initialises a ReleaseHandler.
func New(version string, skipPublish, try bool) *ReleaseHandler {
	// When triggered from CI release branch
	version = strings.TrimPrefix(version, "release-")
	version = strings.TrimPrefix(version, "v")
	rh := &ReleaseHandler{cliVersion: version, skipPublish: skipPublish, try: try}

	if try {
		rh.git = func(args ...string) (string, error) {
			fmt.Println("git", strings.Join(args, " "))
			return "", nil
		}
	} else {
		rh.git = git
	}

	return rh
}

// Run creates a new release.
func (r *ReleaseHandler) Run() error {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return errors.New("GITHUB_TOKEN not set, create one here with the repo scope selected: https://github.com/settings/tokens/new")
	}

	newVersion, finalVersion := r.calculateVersions()

	version := newVersion.String()
	tag := "v" + version

	// Exit early if tag already exists
	exists, err := tagExists(tag)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("Tag %q already exists", tag)
	}

	var changeLogFromTag string

	if newVersion.PatchLevel == 0 {
		// There may have been patch releases between, so set the tag explicitly.
		changeLogFromTag = "v" + newVersion.Prev().String()
		exists, _ := tagExists(changeLogFromTag)
		if !exists {
			// fall back to one that exists.
			changeLogFromTag = ""
		}
	}

	var (
		gitCommits     gitInfos
		gitCommitsDocs gitInfos
		relNotesState  releaseNotesState
	)

	relNotesState, err = r.releaseNotesState(version)
	if err != nil {
		return err
	}

	prepareRelaseNotes := relNotesState == releaseNotesNone
	shouldRelease := relNotesState == releaseNotesReady

	defer r.gitPush() // TODO(bep)

	if prepareRelaseNotes || shouldRelease {
		gitCommits, err = getGitInfos(changeLogFromTag, "hugo", "", !r.try)
		if err != nil {
			return err
		}

		// TODO(bep) explicit tag?
		gitCommitsDocs, err = getGitInfos("", "hugoDocs", "../hugoDocs", !r.try)
		if err != nil {
			return err
		}
	}

	if relNotesState == releaseNotesCreated {
		fmt.Println("Release notes created, but not ready. Reneame to *-ready.md to continue ...")
		return nil
	}

	if prepareRelaseNotes {
		releaseNotesFile, err := r.writeReleaseNotesToTemp(version, gitCommits, gitCommitsDocs)
		if err != nil {
			return err
		}

		if _, err := r.git("add", releaseNotesFile); err != nil {
			return err
		}
		if _, err := r.git("commit", "-m", fmt.Sprintf("%s Add release notes draft for %s\n\nRename to *-ready.md to continue. [ci skip]", commitPrefix, newVersion)); err != nil {
			return err
		}
	}

	if !shouldRelease {
		fmt.Printf("Skip release ... ")
		return nil
	}

	// For docs, for now we assume that:
	// The /docs subtree is up to date and ready to go.
	// The hugoDocs/dev and hugoDocs/master must be merged manually after release.
	// TODO(bep) improve this when we see how it works.

	if err := r.bumpVersions(newVersion); err != nil {
		return err
	}

	if _, err := r.git("commit", "-a", "-m", fmt.Sprintf("%s Bump versions for release of %s\n\n[ci skip]", commitPrefix, newVersion)); err != nil {
		return err
	}

	releaseNotesFile := getReleaseNotesDocsTempFilename(version, true)

	// Write the release notes to the docs site as well.
	docFile, err := r.writeReleaseNotesToDocs(version, releaseNotesFile)
	if err != nil {
		return err
	}

	if _, err := r.git("add", docFile); err != nil {
		return err
	}
	if _, err := r.git("commit", "-m", fmt.Sprintf("%s Add release notes to /docs for release of %s\n\n[ci skip]", commitPrefix, newVersion)); err != nil {
		return err
	}

	if _, err := r.git("tag", "-a", tag, "-m", fmt.Sprintf("%s %s [ci skip]", commitPrefix, newVersion)); err != nil {
		return err
	}

	if !r.skipPublish {
		if _, err := r.git("push", "origin", tag); err != nil {
			return err
		}
	}

	if err := r.release(releaseNotesFile); err != nil {
		return err
	}

	if err := r.bumpVersions(finalVersion); err != nil {
		return err
	}

	if !r.try {
		// No longer needed.
		if err := os.Remove(releaseNotesFile); err != nil {
			return err
		}
	}

	if _, err := r.git("commit", "-a", "-m", fmt.Sprintf("%s Prepare repository for %s\n\n[ci skip]", commitPrefix, finalVersion)); err != nil {
		return err
	}

	return nil
}

func (r *ReleaseHandler) gitPush() {
	if r.skipPublish {
		return
	}
	if _, err := r.git("push", "origin", "HEAD"); err != nil {
		log.Fatal("push failed:", err)
	}
}

func (r *ReleaseHandler) release(releaseNotesFile string) error {
	if r.try {
		fmt.Println("Skip goreleaser...")
		return nil
	}

	cmd := exec.Command("goreleaser", "--rm-dist", "--release-notes", releaseNotesFile, "--skip-publish="+fmt.Sprint(r.skipPublish))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("goreleaser failed: %s", err)
	}
	return nil
}

func (r *ReleaseHandler) bumpVersions(ver helpers.HugoVersion) error {
	toDev := ""

	if ver.Suffix != "" {
		toDev = ver.Suffix
	}

	if err := r.replaceInFile("helpers/hugo.go",
		`Number:(\s{4,})(.*),`, fmt.Sprintf(`Number:${1}%.2f,`, ver.Number),
		`PatchLevel:(\s*)(.*),`, fmt.Sprintf(`PatchLevel:${1}%d,`, ver.PatchLevel),
		`Suffix:(\s{4,})".*",`, fmt.Sprintf(`Suffix:${1}"%s",`, toDev)); err != nil {
		return err
	}

	snapcraftGrade := "stable"
	if ver.Suffix != "" {
		snapcraftGrade = "devel"
	}
	if err := r.replaceInFile("snapcraft.yaml",
		`version: "(.*)"`, fmt.Sprintf(`version: "%s"`, ver),
		`grade: (.*) #`, fmt.Sprintf(`grade: %s #`, snapcraftGrade)); err != nil {
		return err
	}

	var minVersion string
	if ver.Suffix != "" {
		// People use the DEV version in daily use, and we cannot create new themes
		// with the next version before it is released.
		minVersion = ver.Prev().String()
	} else {
		minVersion = ver.String()
	}

	if err := r.replaceInFile("commands/new.go",
		`min_version = "(.*)"`, fmt.Sprintf(`min_version = "%s"`, minVersion)); err != nil {
		return err
	}

	// docs/config.toml
	if err := r.replaceInFile("docs/config.toml",
		`release = "(.*)"`, fmt.Sprintf(`release = "%s"`, ver)); err != nil {
		return err
	}

	return nil
}

func (r *ReleaseHandler) replaceInFile(filename string, oldNew ...string) error {
	fullFilename := hugoFilepath(filename)
	fi, err := os.Stat(fullFilename)
	if err != nil {
		return err
	}

	if r.try {
		fmt.Printf("Replace in %q: %q\n", filename, oldNew)
		return nil
	}

	b, err := ioutil.ReadFile(fullFilename)
	if err != nil {
		return err
	}
	newContent := string(b)

	for i := 0; i < len(oldNew); i += 2 {
		re := regexp.MustCompile(oldNew[i])
		newContent = re.ReplaceAllString(newContent, oldNew[i+1])
	}

	return ioutil.WriteFile(fullFilename, []byte(newContent), fi.Mode())
}

func hugoFilepath(filename string) string {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(pwd, filename)
}

func isCI() bool {
	return os.Getenv("CI") != ""
}

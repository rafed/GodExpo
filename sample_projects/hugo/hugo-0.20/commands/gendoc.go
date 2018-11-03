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
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/hugo/helpers"
	"github.com/spf13/hugo/hugofs"
	jww "github.com/spf13/jwalterweatherman"
)

const gendocFrontmatterTemplate = `---
date: %s
title: "%s"
slug: %s
url: %s
---
`

var gendocdir string
var gendocCmd = &cobra.Command{
	Use:   "doc",
	Short: "Generate Markdown documentation for the Hugo CLI.",
	Long: `Generate Markdown documentation for the Hugo CLI.

This command is, mostly, used to create up-to-date documentation
of Hugo's command-line interface for http://gohugo.io/.

It creates one Markdown file per command with front matter suitable
for rendering in Hugo.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		if !strings.HasSuffix(gendocdir, helpers.FilePathSeparator) {
			gendocdir += helpers.FilePathSeparator
		}
		if found, _ := helpers.Exists(gendocdir, hugofs.Os); !found {
			jww.FEEDBACK.Println("Directory", gendocdir, "does not exist, creating...")
			if err := hugofs.Os.MkdirAll(gendocdir, 0777); err != nil {
				return err
			}
		}
		now := time.Now().Format(time.RFC3339)
		prepender := func(filename string) string {
			name := filepath.Base(filename)
			base := strings.TrimSuffix(name, path.Ext(name))
			url := "/commands/" + strings.ToLower(base) + "/"
			return fmt.Sprintf(gendocFrontmatterTemplate, now, strings.Replace(base, "_", " ", -1), base, url)
		}

		linkHandler := func(name string) string {
			base := strings.TrimSuffix(name, path.Ext(name))
			return "/commands/" + strings.ToLower(base) + "/"
		}

		jww.FEEDBACK.Println("Generating Hugo command-line documentation in", gendocdir, "...")
		doc.GenMarkdownTreeCustom(cmd.Root(), gendocdir, prepender, linkHandler)
		jww.FEEDBACK.Println("Done.")

		return nil
	},
}

func init() {
	gendocCmd.PersistentFlags().StringVar(&gendocdir, "dir", "/tmp/hugodoc/", "the directory to write the doc.")

	// For bash-completion
	gendocCmd.PersistentFlags().SetAnnotation("dir", cobra.BashCompSubdirsInDir, []string{})
}

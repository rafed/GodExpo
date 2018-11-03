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

package strings

import (
	"github.com/gohugoio/hugo/deps"
	"github.com/gohugoio/hugo/tpl/internal"
)

const name = "strings"

func init() {
	f := func(d *deps.Deps) *internal.TemplateFuncsNamespace {
		ctx := New(d)

		ns := &internal.TemplateFuncsNamespace{
			Name:    name,
			Context: func(args ...interface{}) interface{} { return ctx },
		}

		ns.AddMethodMapping(ctx.Chomp,
			[]string{"chomp"},
			[][2]string{
				{`{{chomp "<p>Blockhead</p>\n" }}`, `<p>Blockhead</p>`},
			},
		)

		ns.AddMethodMapping(ctx.CountRunes,
			[]string{"countrunes"},
			[][2]string{},
		)

		ns.AddMethodMapping(ctx.CountWords,
			[]string{"countwords"},
			[][2]string{},
		)

		ns.AddMethodMapping(ctx.FindRE,
			[]string{"findRE"},
			[][2]string{
				{
					`{{ findRE "[G|g]o" "Hugo is a static side generator written in Go." "1" }}`,
					`[go]`},
			},
		)

		ns.AddMethodMapping(ctx.HasPrefix,
			[]string{"hasPrefix"},
			[][2]string{
				{`{{ hasPrefix "Hugo" "Hu" }}`, `true`},
				{`{{ hasPrefix "Hugo" "Fu" }}`, `false`},
			},
		)

		ns.AddMethodMapping(ctx.ToLower,
			[]string{"lower"},
			[][2]string{
				{`{{lower "BatMan"}}`, `batman`},
			},
		)

		ns.AddMethodMapping(ctx.Replace,
			[]string{"replace"},
			[][2]string{
				{
					`{{ replace "Batman and Robin" "Robin" "Catwoman" }}`,
					`Batman and Catwoman`},
			},
		)

		ns.AddMethodMapping(ctx.ReplaceRE,
			[]string{"replaceRE"},
			[][2]string{},
		)

		ns.AddMethodMapping(ctx.SliceString,
			[]string{"slicestr"},
			[][2]string{
				{`{{slicestr "BatMan" 0 3}}`, `Bat`},
				{`{{slicestr "BatMan" 3}}`, `Man`},
			},
		)

		ns.AddMethodMapping(ctx.Split,
			[]string{"split"},
			[][2]string{},
		)

		ns.AddMethodMapping(ctx.Substr,
			[]string{"substr"},
			[][2]string{
				{`{{substr "BatMan" 0 -3}}`, `Bat`},
				{`{{substr "BatMan" 3 3}}`, `Man`},
			},
		)

		ns.AddMethodMapping(ctx.Trim,
			[]string{"trim"},
			[][2]string{
				{`{{ trim "++Batman--" "+-" }}`, `Batman`},
			},
		)

		ns.AddMethodMapping(ctx.Title,
			[]string{"title"},
			[][2]string{
				{`{{title "Bat man"}}`, `Bat Man`},
			},
		)

		ns.AddMethodMapping(ctx.Truncate,
			[]string{"truncate"},
			[][2]string{
				{`{{ "this is a very long text" | truncate 10 " ..." }}`, `this is a ...`},
				{`{{ "With [Markdown](/markdown) inside." | markdownify | truncate 14 }}`, `With <a href="/markdown">Markdown …</a>`},
			},
		)

		ns.AddMethodMapping(ctx.ToUpper,
			[]string{"upper"},
			[][2]string{
				{`{{upper "BatMan"}}`, `BATMAN`},
			},
		)

		return ns

	}

	internal.AddTemplateFuncsNamespace(f)
}

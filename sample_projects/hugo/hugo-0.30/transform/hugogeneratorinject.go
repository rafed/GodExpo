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

package transform

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/gohugoio/hugo/helpers"
)

var metaTagsCheck = regexp.MustCompile(`(?i)<meta\s+name=['|"]?generator['|"]?`)
var hugoGeneratorTag = fmt.Sprintf(`<meta name="generator" content="Hugo %s" />`, helpers.CurrentHugoVersion)

// HugoGeneratorInject injects a meta generator tag for Hugo if none present.
func HugoGeneratorInject(ct contentTransformer) {
	if metaTagsCheck.Match(ct.Content()) {
		if _, err := ct.Write(ct.Content()); err != nil {
			helpers.DistinctWarnLog.Println("Failed to inject Hugo generator tag:", err)
		}
		return
	}

	head := "<head>"
	replace := []byte(fmt.Sprintf("%s\n\t%s", head, hugoGeneratorTag))
	newcontent := bytes.Replace(ct.Content(), []byte(head), replace, 1)

	if len(newcontent) == len(ct.Content()) {
		head := "<HEAD>"
		replace := []byte(fmt.Sprintf("%s\n\t%s", head, hugoGeneratorTag))
		newcontent = bytes.Replace(ct.Content(), []byte(head), replace, 1)
	}

	if _, err := ct.Write(newcontent); err != nil {
		helpers.DistinctWarnLog.Println("Failed to inject Hugo generator tag:", err)
	}

}

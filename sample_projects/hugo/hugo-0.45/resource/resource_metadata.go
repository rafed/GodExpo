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
	"strconv"

	"github.com/spf13/cast"

	"strings"

	"github.com/gohugoio/hugo/common/maps"
)

var (
	_ metaAssigner = (*genericResource)(nil)
)

// metaAssigner allows updating metadata in resources that supports it.
type metaAssigner interface {
	setTitle(title string)
	setName(name string)
	updateParams(params map[string]interface{})
}

const counterPlaceHolder = ":counter"

// AssignMetadata assigns the given metadata to those resources that supports updates
// and matching by wildcard given in `src` using `filepath.Match` with lower cased values.
// This assignment is additive, but the most specific match needs to be first.
// The `name` and `title` metadata field support shell-matched collection it got a match in.
// See https://golang.org/pkg/path/#Match
func AssignMetadata(metadata []map[string]interface{}, resources ...Resource) error {

	counters := make(map[string]int)

	for _, r := range resources {
		if _, ok := r.(metaAssigner); !ok {
			continue
		}

		var (
			nameSet, titleSet                   bool
			nameCounter, titleCounter           = 0, 0
			nameCounterFound, titleCounterFound bool
			resourceSrcKey                      = strings.ToLower(r.Name())
		)

		ma := r.(metaAssigner)
		for _, meta := range metadata {
			src, found := meta["src"]
			if !found {
				return fmt.Errorf("missing 'src' in metadata for resource")
			}

			srcKey := strings.ToLower(cast.ToString(src))

			glob, err := getGlob(srcKey)
			if err != nil {
				return fmt.Errorf("failed to match resource with metadata: %s", err)
			}

			match := glob.Match(resourceSrcKey)

			if match {
				if !nameSet {
					name, found := meta["name"]
					if found {
						name := cast.ToString(name)
						if !nameCounterFound {
							nameCounterFound = strings.Contains(name, counterPlaceHolder)
						}
						if nameCounterFound && nameCounter == 0 {
							counterKey := "name_" + srcKey
							nameCounter = counters[counterKey] + 1
							counters[counterKey] = nameCounter
						}

						ma.setName(replaceResourcePlaceholders(name, nameCounter))
						nameSet = true
					}
				}

				if !titleSet {
					title, found := meta["title"]
					if found {
						title := cast.ToString(title)
						if !titleCounterFound {
							titleCounterFound = strings.Contains(title, counterPlaceHolder)
						}
						if titleCounterFound && titleCounter == 0 {
							counterKey := "title_" + srcKey
							titleCounter = counters[counterKey] + 1
							counters[counterKey] = titleCounter
						}
						ma.setTitle((replaceResourcePlaceholders(title, titleCounter)))
						titleSet = true
					}
				}

				params, found := meta["params"]
				if found {
					m := cast.ToStringMap(params)
					// Needed for case insensitive fetching of params values
					maps.ToLower(m)
					ma.updateParams(m)
				}
			}
		}
	}

	return nil
}

func replaceResourcePlaceholders(in string, counter int) string {
	return strings.Replace(in, counterPlaceHolder, strconv.Itoa(counter), -1)
}

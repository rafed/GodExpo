---
aliases:
- /layout/functions/
lastmod: 2015-09-20
date: 2013-07-01
linktitle: Functions
toc: true
menu:
  main:
    parent: layout
next: /templates/variables
prev: /templates/go-templates
title: Hugo Template Functions
weight: 20
---

Hugo uses the excellent Go html/template library for its template engine.
It is an extremely lightweight engine that provides a very small amount of
logic. In our experience, it is just the right amount of logic to be able
to create a good static website.

Go templates are lightweight but extensible. Hugo has added the following
functions to the basic template logic.

(Go itself supplies built-in functions, including comparison operators
and other basic tools; these are listed in the
[Go template documentation](http://golang.org/pkg/text/template/#hdr-Functions).)

## General

### default
Checks whether a given value is set and returns a default value if it is not.
"Set" in this context means non-zero for numeric types and times;
non-zero length for strings, arrays, slices, and maps;
any boolean or struct value; or non-nil for any other types.

e.g.

    {{ index .Params "font" | default "Roboto" }} → default is "Roboto"
    {{ default "Roboto" (index .Params "font") }} → default is "Roboto"

### delimit
Loops through any array, slice or map and returns a string of all the values separated by the delimiter. There is an optional third parameter that lets you choose a different delimiter to go between the last two values.
Maps will be sorted by the keys, and only a slice of the values will be returned, keeping a consistent output order.

Works on [lists](/templates/list/), [taxonomies](/taxonomies/displaying/), [terms](/templates/terms/), [groups](/templates/list/)

e.g.

    // Front matter
    +++
    tags: [ "tag1", "tag2", "tag3" ]
    +++

    // Used anywhere in a template
    Tags: {{ delimit .Params.tags ", " }}

    // Outputs Tags: tag1, tag2, tag3

    // Example with the optional "last" parameter
    Tags: {{ delimit .Params.tags ", " " and " }}

    // Outputs Tags: tag1, tag2 and tag3

### dict
Creates a dictionary `(map[string, interface{})`, expects parameters added in value:object fasion.
Invalid combinations like keys that are not strings or uneven number of parameters, will result in an exception thrown.
Useful for passing maps to partials when adding to a template.

e.g. Pass into "foo.html" a map with the keys "important, content"

    {{$important := .Site.Params.SomethingImportant }}
    {{range .Site.Params.Bar}}
        {{partial "foo" (dict "content" . "important" $important)}}
    {{end}}

"foo.html"

    Important {{.important}}
    {{.content}}

or create a map on the fly to pass into

    {{partial "foo" (dict "important" "Smiles" "content" "You should do more")}}



### slice

`slice` allows you to create an array (`[]interface{}`) of all arguments that you pass to this function.

One use case is the concatenation of elements in combination with `delimit`:

```html
{{ delimit (slice "foo" "bar" "buzz") ", " }}
<!-- returns the string "foo, bar, buzz" -->
```


### shuffle

`shuffle` returns a random permutation of a given array or slice, e.g.

```html
{{ shuffle (seq 1 5) }}
<!-- returns [2 5 3 1 4] -->

{{ shuffle (slice "foo" "bar" "buzz") }}
<!-- returns [buzz foo bar] -->
```

### echoParam
Prints a parameter if it is set.

e.g. `{{ echoParam .Params "project_url" }}`


### eq
Returns true if the parameters are equal.

e.g.

    {{ if eq .Section "blog" }}current{{ end }}


### first
Slices an array to only the first _N_ elements.

Works on [lists](/templates/list/), [taxonomies](/taxonomies/displaying/), [terms](/templates/terms/), [groups](/templates/list/)

e.g.

    {{ range first 10 .Data.Pages }}
        {{ .Render "summary" }}
    {{ end }}


### jsonify
Encodes a given object to JSON.

e.g.

    {{ dict "title" .Title "content" .Plain | jsonify }}

### last
Slices an array to only the last _N_ elements.

Works on [lists](/templates/list/), [taxonomies](/taxonomies/displaying/), [terms](/templates/terms/), [groups](/templates/list/)

e.g.

    {{ range last 10 .Data.Pages }}
        {{ .Render "summary" }}
    {{ end }}

### after
Slices an array to only the items after the <em>N</em>th item. Use this in combination
with `first` to use both halves of an array split at item _N_.

Works on [lists](/templates/list/), [taxonomies](/taxonomies/displaying/), [terms](/templates/terms/), [groups](/templates/list/)

e.g.

    {{ range after 10 .Data.Pages }}
        {{ .Render "title" }}
    {{ end }}

### getenv
Returns the value of an environment variable.

Takes a string containing the name of the variable as input. Returns
an empty string if the variable is not set, otherwise returns the
value of the variable. Note that in Unix-like environments, the
variable must also be exported in order to be seen by `hugo`.

e.g.

    {{ getenv "HOME" }}


### in
Checks if an element is in an array (or slice) and returns a boolean.
The elements supported are strings, integers and floats (only float64 will match as expected).
In addition, it can also check if a substring exists in a string.

e.g.

    {{ if in .Params.tags "Git" }}Follow me on GitHub!{{ end }}

or

    {{ if in "this string contains a substring" "substring" }}Substring found!{{ end }}


### intersect
Given two arrays (or slices), this function will return the common elements in the arrays.
The elements supported are strings, integers and floats (only float64).

A useful example of this functionality is a 'similar posts' block.
Create a list of links to posts where any of the tags in the current post match any tags in other posts.

e.g.

    <ul>
    {{ $page_link := .Permalink }}
    {{ $tags := .Params.tags }}
    {{ range .Site.Pages }}
        {{ $page := . }}
        {{ $has_common_tags := intersect $tags .Params.tags | len | lt 0 }}
        {{ if and $has_common_tags (ne $page_link $page.Permalink) }}
            <li><a href="{{ $page.Permalink }}">{{ $page.Title }}</a></li>
        {{ end }}
    {{ end }}
    </ul>


### union
Given two arrays (or slices) A and B, this function will return a new array that contains the elements or objects that belong to either A or to B or to both. The elements supported are strings, integers and floats (only float64).

```
{{ union (slice 1 2 3) (slice 3 4 5) }}
<!-- returns [1 2 3 4 5] -->

{{ union (slice 1 2 3) nil }}
<!-- returns [1 2 3] -->

{{ union nil (slice 1 2 3) }}
<!-- returns [1 2 3] -->

{{ union nil nil }}
<!-- returns an error because both arrays/slices have to be of the same type -->
```

### isset
Returns true if the parameter is set.
Takes either a slice, array or channel and an index or a map and a key as input.

e.g. `{{ if isset .Params "project_url" }} {{ index .Params "project_url" }}{{ end }}`

### seq

Creates a sequence of integers. It's named and used as GNU's seq.

Some examples:

* `3` => `1, 2, 3`
* `1 2 4` => `1, 3`
* `-3` => `-1, -2, -3`
* `1 4` => `1, 2, 3, 4`
* `1 -2` => `1, 0, -1, -2`

### sort
Sorts maps, arrays and slices, returning a sorted slice.
A sorted array of map values will be returned, with the keys eliminated.
There are two optional arguments, which are `sortByField` and `sortAsc`.
If left blank, sort will sort by keys (for maps) in ascending order.

Works on [lists](/templates/list/), [taxonomies](/taxonomies/displaying/), [terms](/templates/terms/), [groups](/templates/list/)

e.g.

    // Front matter
    +++
    tags: [ "tag3", "tag1", "tag2" ]
    +++

    // Site config
    +++
    [params.authors]
      [params.authors.Derek]
        "firstName"  = "Derek"
        "lastName"   = "Perkins"
      [params.authors.Joe]
        "firstName"  = "Joe"
        "lastName"   = "Bergevin"
      [params.authors.Tanner]
        "firstName"  = "Tanner"
        "lastName"   = "Linsley"
    +++

    // Use default sort options - sort by key / ascending
    Tags: {{ range sort .Params.tags }}{{ . }} {{ end }}

    // Outputs Tags: tag1 tag2 tag3

    // Sort by value / descending
    Tags: {{ range sort .Params.tags "value" "desc" }}{{ . }} {{ end }}

    // Outputs Tags: tag3 tag2 tag1

    // Use default sort options - sort by value / descending
    Authors: {{ range sort .Site.Params.authors }}{{ .firstName }} {{ end }}

    // Outputs Authors: Derek Joe Tanner

    // Use default sort options - sort by value / descending
    Authors: {{ range sort .Site.Params.authors "lastName" "desc" }}{{ .lastName }} {{ end }}

    // Outputs Authors: Perkins Linsley Bergevin


### where
Filters an array to only elements containing a matching value for a given field.

Works on [lists](/templates/list/), [taxonomies](/taxonomies/displaying/), [terms](/templates/terms/), [groups](/templates/list/)

e.g.

    {{ range where .Data.Pages "Section" "post" }}
       {{ .Content }}
    {{ end }}

It can be used with dot chaining second argument to refer a nested element of a value.

e.g.

    // Front matter on some pages
    +++
    series: golang
    +++

    {{ range where .Site.Pages "Params.series" "golang" }}
       {{ .Content }}
    {{ end }}

It can also be used with an operator like `!=`, `>=`, `in` etc. Without an operator (like above), `where` compares a given field with a matching value in a way like `=` is specified.

e.g.

    {{ range where .Data.Pages "Section" "!=" "post" }}
       {{ .Content }}
    {{ end }}

Following operators are now available

- `=`, `==`, `eq`: True if a given field value equals a matching value
- `!=`, `<>`, `ne`: True if a given field value doesn't equal a matching value
- `>=`, `ge`: True if a given field value is greater than or equal to a matching value
- `>`, `gt`: True if a given field value is greater than a matching value
- `<=`, `le`: True if a given field value is lesser than or equal to a matching value
- `<`, `lt`: True if a given field value is lesser than a matching value
- `in`: True if a given field value is included in a matching value. A matching value must be an array or a slice
- `not in`: True if a given field value isn't included in a matching value. A matching value must be an array or a slice
- `intersect`: True if a given field value that is a slice / array of strings or integers contains elements in common with the matching value. It follows the same rules as the intersect function.

*`intersect` operator, e.g.:*

    {{ range where .Site.Pages ".Params.tags" "intersect" .Params.tags }}
      {{ if ne .Permalink $.Permalink }}
        {{ .Render "summary" }}
      {{ end }}
    {{ end }}

*`where` and `first` can be stacked, e.g.:*

    {{ range first 5 (where .Data.Pages "Section" "post") }}
       {{ .Content }}
    {{ end }}

### Unset field
Filter only work for set fields. To check whether a field is set or exist, use operand `nil`.

This can be useful to filter a small amount of pages from a large pool. Instead of set field on all pages, you can set field on required pages only.

Only following operators are available for `nil`

- `=`, `==`, `eq`: True if the given field is not set.
- `!=`, `<>`, `ne`: True if the given field is set.

e.g.

    {{ range where .Data.Pages ".Params.specialpost" "!=" nil }}
       {{ .Content }}
    {{ end }}

## Files

### readDir

Gets a directory listing from a directory relative to the current project working dir.

So, If the project working dir has a single file named `README.txt`:

`{{ range (readDir ".") }}{{ .Name }}{{ end }}` → "README.txt"

### readFile
Reads a file from disk and converts it into a string. Note that the filename must be relative to the current project working dir.
 So, if you have a file with the name `README.txt` in the root of your project with the content `Hugo Rocks!`:

 `{{readFile "README.txt"}}` → `"Hugo Rocks!"`

### imageConfig
Parses the image and returns the height, width and color model.

e.g.
```
{{ with (imageConfig "favicon.ico") }}
favicon.ico: {{.Width}} x {{.Height}}
{{ end }}
```

## Math

<table class="table table-bordered">
<thead>
<tr>
<th>Function</th>
<th>Description</th>
<th>Example</th>
</tr>
</thead>

<tbody>
<tr>
<td><code>add</code></td>
<td>Adds two integers.</td>
<td><code>{{add 1 2}}</code> → 3</td>
</tr>

<tr>
<td><code>div</code></td>
<td>Divides two integers.</td>
<td><code>{{div 6 3}}</code> → 2</td>
</tr>

<tr>
<td><code>mod</code></td>
<td>Modulus of two integers.</td>
<td><code>{{mod 15 3}}</code> → 0</td>
</tr>

<tr>
<td><code>modBool</code></td>
<td>Boolean of modulus of two integers.  <code>true</code> if modulus is 0.</td>
<td><code>{{modBool 15 3}}</code> → true</td>
</tr>

<tr>
<td><code>mul</code></td>
<td>Multiplies two integers.</td>
<td><code>{{mul 2 3}}</code> → 6</td>
</tr>

<tr>
<td><code>sub</code></td>
<td>Subtracts two integers.</td>
<td><code>{{sub 3 2}}</code> → 1</td>
</tr>

</tbody>
</table>

## Numbers

### int

Creates an `int`.

e.g.

* `{{ int "123" }}` → 123

## Strings

### printf

Format a string using the standard `fmt.Sprintf` function. See [the go
doc](https://golang.org/pkg/fmt/) for reference.
A
e.g., `{{ i18n ( printf "combined_%s" $var ) }}` or `{{ printf "formatted %.2f" 3.1416 }}`

### chomp
Removes any trailing newline characters. Useful in a pipeline to remove newlines added by other processing (including `markdownify`).

e.g., `{{chomp "<p>Blockhead</p>\n"}}` → `"<p>Blockhead</p>"`


### dateFormat
Converts the textual representation of the datetime into the other form or returns it of Go `time.Time` type value.
These are formatted with the layout string.

e.g. `{{ dateFormat "Monday, Jan 2, 2006" "2015-01-21" }}` → "Wednesday, Jan 21, 2015"


### emojify

Runs the string through the Emoji emoticons processor. The result will be declared as "safe" so Go templates will not filter it.

See the [Emoji cheat sheet](http://www.emoji-cheat-sheet.com/) for available emoticons.

e.g. `{{ "I :heart: Hugo" | emojify }}`

### highlight
Takes a string of code and a language, uses Pygments to return the syntax highlighted code in HTML.
Used in the [highlight shortcode](/extras/highlighting/).

### htmlEscape
HtmlEscape returns the given string with the critical reserved HTML codes escaped,
such that `&` becomes `&amp;` and so on. It escapes only: `<`, `>`, `&`, `'` and `"`.

Bear in mind that, unless content is passed to `safeHTML`, output strings are escaped
usually by the processor anyway.

e.g.
`{{ htmlEscape "Hugo & Caddy > Wordpress & Apache" }} → "Hugo &amp; Caddy &gt; Wordpress &amp; Apache"`

### htmlUnescape
HtmlUnescape returns the given string with html escape codes un-escaped. This
un-escapes more codes than `htmlEscape` escapes, including `#` codes and pre-UTF8
escapes for accented characters. It defers completely to the Go `html.UnescapeString`
function, so functionality is consistent with that codebase.

Remember to pass the output of this to `safeHTML` if fully unescaped characters
are desired, or the output will be escaped again as normal.

e.g.
`{{ htmlUnescape "Hugo &amp; Caddy &gt; Wordpress &amp; Apache" }} → "Hugo & Caddy > Wordpress & Apache"`

### humanize
Humanize returns the humanized version of an argument with the first letter capitalized.
If the input is either an int64 value or the string representation of an integer, humanize returns the number with the proper ordinal appended.

e.g.
```
{{humanize "my-first-post"}} → "My first post"
{{humanize "myCamelPost"}} → "My camel post"
{{humanize "52"}} → "52nd"
{{humanize 103}} → "103rd"
```


### lower
Converts all characters in string to lowercase.

e.g. `{{lower "BatMan"}}` → "batman"


### markdownify

Runs the string through the Markdown processor. The result will be declared as "safe" so Go templates will not filter it.

e.g. `{{ .Title | markdownify }}`

### plainify

Strips any HTML and returns the plain text version.

e.g. `{{ "<b>BatMan</b>" | plainify }}` → "BatMan"

### pluralize
Pluralize the given word with a set of common English pluralization rules.

e.g. `{{ "cat" | pluralize }}` → "cats"

### findRE
Returns a list of strings that match the regular expression. By default all matches will be included. The number of matches can be limited with an optional third parameter.

The example below returns a list of all second level headers (`<h2>`) in the content:

    {{ findRE "<h2.*?>(.|\n)*?</h2>" .Content }}

We can limit the number of matches in that list with a third parameter. Let's say we want to have at most one match (or none if no substring matched):

    {{ findRE "<h2.*?>(.|\n)*?</h2>" .Content 1 }}
    <!-- returns ["<h2 id="#foo">Foo</h2>"] -->

`findRE` allows us to build an automatically generated table of contents that could be used for a simple scrollspy:

    {{ $headers := findRE "<h2.*?>(.|\n)*?</h2>" .Content }}

    {{ if ge (len $headers) 1 }}
        <ul>
        {{ range $headers }}
            <li>
                <a href="#{{ . | plainify | urlize }}">
                    {{ . | plainify }}
                </a>
            </li>
        {{ end }}
        </ul>
    {{ end }}

First, we try to find all second-level headers and generate a list if at least one header was found. `plainify` strips the HTML and `urlize` converts the header into a valid URL.

### replace
Replaces all occurrences of the search string with the replacement string.

e.g. `{{ replace "Batman and Robin" "Robin" "Catwoman" }}` → "Batman and Catwoman"


### replaceRE
Replaces all occurrences of a regular expression with the replacement pattern.

e.g. `{{ replaceRE "^https?://([^/]+).*" "$1" "http://gohugo.io/docs" }}` → "gohugo.io"
e.g. `{{ "http://gohugo.io/docs" | replaceRE "^https?://([^/]+).*" "$1" }}` → "gohugo.io"


### safeHTML
Declares the provided string as a "safe" HTML document fragment
so Go html/template will not filter it.  It should not be used
for HTML from a third-party, or HTML with unclosed tags or comments.

Example: Given a site-wide `config.toml` that contains this line:

    copyright = "© 2015 Jane Doe.  <a href=\"http://creativecommons.org/licenses/by/4.0/\">Some rights reserved</a>."

`{{ .Site.Copyright | safeHTML }}` would then output:

> © 2015 Jane Doe.  <a href="http://creativecommons.org/licenses/by/4.0/">Some rights reserved</a>.

However, without the `safeHTML` function, html/template assumes
`.Site.Copyright` to be unsafe, escaping all HTML tags,
rendering the whole string as plain-text like this:

<blockquote>
<p>© 2015 Jane Doe.  &lt;a href=&#34;http://creativecommons.org/licenses/by/4.0/&#34;&gt;Some rights reserved&lt;/a&gt;.</p>
</blockquote>

### safeHTMLAttr
Declares the provided string as a "safe" HTML attribute
from a trusted source, for example, ` dir="ltr"`,
so Go html/template will not filter it.

Example: Given a site-wide `config.toml` that contains this menu entry:

    [[menu.main]]
        name = "IRC: #golang at freenode"
        url = "irc://irc.freenode.net/#golang"

* `<a href="{{ .URL }}">` ⇒ `<a href="#ZgotmplZ">` (Bad!)
* `<a {{ printf "href=%q" .URL | safeHTMLAttr }}>` ⇒ `<a href="irc://irc.freenode.net/#golang">` (Good!)

### safeCSS
Declares the provided string as a known "safe" CSS string
so Go html/templates will not filter it.
"Safe" means CSS content that matches any of:

1. The CSS3 stylesheet production, such as `p { color: purple }`.
2. The CSS3 rule production, such as `a[href=~"https:"].foo#bar`.
3. CSS3 declaration productions, such as `color: red; margin: 2px`.
4. The CSS3 value production, such as `rgba(0, 0, 255, 127)`.

Example: Given `style = "color: red;"` defined in the front matter of your `.md` file:

* `<p style="{{ .Params.style | safeCSS }}">…</p>` ⇒ `<p style="color: red;">…</p>` (Good!)
* `<p style="{{ .Params.style }}">…</p>` ⇒ `<p style="ZgotmplZ">…</p>` (Bad!)

Note: "ZgotmplZ" is a special value that indicates that unsafe content reached a
CSS or URL context.

### safeJS

Declares the provided string as a known "safe" Javascript string so Go
html/templates will not escape it.  "Safe" means the string encapsulates a known
safe EcmaScript5 Expression, for example, `(x + y * z())`. Template authors
are responsible for ensuring that typed expressions do not break the intended
precedence and that there is no statement/expression ambiguity as when passing
an expression like `{ foo:bar() }\n['foo']()`, which is both a valid Expression
and a valid Program with a very different meaning.

Example: Given `hash = "619c16f"` defined in the front matter of your `.md` file:

* `<script>var form_{{ .Params.hash | safeJS }};…</script>` ⇒ `<script>var form_619c16f;…</script>` (Good!)
* `<script>var form_{{ .Params.hash }};…</script>` ⇒ `<script>var form_"619c16f";…</script>` (Bad!)

### singularize
Singularize the given word with a set of common English singularization rules.

e.g. `{{ "cats" | singularize }}` → "cat"

### slicestr

Slicing in `slicestr` is done by specifying a half-open range with two indices, `start` and `end`.
For example, 1 and 4 creates a slice including elements 1 through 3.
The `end` index can be omitted; it defaults to the string's length.

e.g.

* `{{slicestr "BatMan" 3}}` → "Man"
* `{{slicestr "BatMan" 0 3}}` → "Bat"

### truncate

Truncate a text to a max length without cutting words or leaving unclosed HTML tags. Since Go templates are HTML-aware, truncate will handle normal strings vs HTML strings intelligently. It's important to note that if you have a raw string that contains HTML tags that you want treated as HTML, you will need to convert the string to HTML using the safeHTML template function before sending the value to truncate; otherwise, the HTML tags will be escaped by truncate.

e.g.

* `{{ "this is a text" | truncate 10 " ..." }}` → `this is a ...`
* `{{ "<em>Keep my HTML</em>" | safeHTML | truncate 10 }}` → `<em>Keep my …</em>`
* `{{ "With [Markdown](#markdown) inside." | markdownify | truncate 10 }}` → `With <a href='#markdown'>Markdown …</a>`  

### split

Split a string into substrings separated by a delimiter.

e.g.

* `{{split "tag1,tag2,tag3" "," }}` → ["tag1" "tag2" "tag3"]

### string

Creates a `string`.

e.g.

* `{{string "BatMan"}}` → "BatMan"

### substr

Extracts parts of a string, beginning at the character at the specified
position, and returns the specified number of characters.

It normally takes two parameters: `start` and `length`.
It can also take one parameter: `start`, i.e. `length` is omitted, in which case
the substring starting from start until the end of the string will be returned.

To extract characters from the end of the string, use a negative start number.

In addition, borrowing from the extended behavior described at http://php.net/substr,
if `length` is given and is negative, then that many characters will be omitted from
the end of string.

e.g.

* `{{substr "BatMan" 0 -3}}` → "Bat"
* `{{substr "BatMan" 3 3}}` → "Man"

### hasPrefix

HasPrefix tests whether a string begins with prefix.

* `{{ hasPrefix "Hugo" "Hu" }}` → true

### title
Converts all characters in string to titlecase.

e.g. `{{title "BatMan"}}` → "Batman"


### trim
Returns a slice of the string with all leading and trailing characters contained in cutset removed.

e.g. `{{ trim "++Batman--" "+-" }}` → "Batman"


### upper
Converts all characters in string to uppercase.

e.g. `{{upper "BatMan"}}` → "BATMAN"


### countwords

`countwords` tries to convert the passed content to a string and counts each word
in it. The template functions works similar to [.WordCount]({{< relref "templates/variables.md#page-variables" >}}).

```html
{{ "Hugo is a static site generator." | countwords }}
<!-- outputs a content length of 6 words.  -->
```


### countrunes

Alternatively to counting all words , `countrunes` determines the number  of runes in the content and excludes any whitespace. This can become useful if you have to deal with
CJK-like languages.

```html
{{ "Hello, 世界" | countrunes }}
<!-- outputs a content length of 8 runes. -->
```

### md5

`md5` hashes the given input and returns its MD5 checksum.

```html
{{ md5 "Hello world, gophers!" }}
<!-- returns the string "b3029f756f98f79e7f1b7f1d1f0dd53b" -->
```

This can be useful if you want to use Gravatar for generating a unique avatar:

```html
<img src="https://www.gravatar.com/avatar/{{ md5 "your@email.com" }}?s=100&d=identicon">
```


### sha1

`sha1` hashes the given input and returns its SHA1 checksum.

```html
{{ sha1 "Hello world, gophers!" }}
<!-- returns the string "c8b5b0e33d408246e30f53e32b8f7627a7a649d4" -->
```


### sha256

`sha256` hashes the given input and returns its SHA256 checksum.

```html
{{ sha256 "Hello world, gophers!" }}
<!-- returns the string "6ec43b78da9669f50e4e422575c54bf87536954ccd58280219c393f2ce352b46" -->
```


## Internationalization

### i18n

This translates a piece of content based on your `i18n/en-US.yaml`
(and friends) files. You can use the [go-i18n](https://github.com/nicksnyder/go-i18n) tools to manage your translations.  The translations can exist in both the theme and at the root of your repository.

e.g.: `{{ i18n "translation_id" }}`

For more information about string translations, see [Translation of strings]({{< relref "content/multilingual.md#translation-of-strings">}}).

### T

`T` is an alias to `i18n`. E.g. `{{ T "translation_id" }}`.

## Times

### time

`time` converts a timestamp string into a [`time.Time`](https://godoc.org/time#Time) structure so you can access its fields. E.g.

* `{{ time "2016-05-28" }}` → "2016-05-28T00:00:00Z"
* `{{ (time "2016-05-28").YearDay }}` → 149
* `{{ mul 1000 (time "2016-05-28T10:30:00.00+10:00").Unix }}` → 1464395400000 (Unix time in milliseconds)

### now

`now` returns the current local time as a [`time.Time`](https://godoc.org/time#Time).

## URLs
### absLangURL, relLangURL
These are similar to the `absURL` and `relURL` relatives below, but will add the correct language prefix when the site is configured with more than one language.

So for a site  `baseURL` set to `http://mysite.com/hugo/` and the current language is `en`:

* `{{ "blog/" | absLangURL }}` → "http://mysite.com/hugo/en/blog/"
* `{{ "blog/" | relLangURL }}` → "/hugo/en/blog/"

### absURL, relURL

Both `absURL` and `relURL` considers the configured value of `baseURL`, so given a `baseURL` set to `http://mysite.com/hugo/`:

* `{{ "mystyle.css" | absURL }}` → "http://mysite.com/hugo/mystyle.css"
* `{{ "mystyle.css" | relURL }}` → "/hugo/mystyle.css"
* `{{ "http://gohugo.io/" | relURL }}` →  "http://gohugo.io/"
* `{{ "http://gohugo.io/" | absURL }}` →  "http://gohugo.io/"

The last two examples may look funky, but is useful if you, say, have a list of images, some of them hosted externally, some locally:

```
<script type="application/ld+json">
{
    "@context" : "http://schema.org",
    "@type" : "BlogPosting",
    "image" : {{ apply .Params.images "absURL" "." }}
}
</script>
```

The above also exploits the fact that the Go template parser JSON-encodes objects inside `script` tags.



**Note:** These functions are smart about missing slashes, but will not add one to the end if not present.


### ref, relref
Looks up a content page by relative path or logical name to return the permalink (`ref`) or relative permalink (`relref`). Requires a `Page` object (usually satisfied with `.`). Used in the [`ref` and `relref` shortcodes]({{% ref "extras/crossreferences.md" %}}).

e.g. {{ ref . "about.md" }}

### safeURL
Declares the provided string as a "safe" URL or URL substring (see [RFC 3986][]).
A URL like `javascript:checkThatFormNotEditedBeforeLeavingPage()` from a trusted
source should go in the page, but by default dynamic `javascript:` URLs are
filtered out since they are a frequently exploited injection vector.

[RFC 3986]: http://tools.ietf.org/html/rfc3986

Without `safeURL`, only the URI schemes `http:`, `https:` and `mailto:`
are considered safe by Go.  If any other URI schemes, e.g.&nbsp;`irc:` and
`javascript:`, are detected, the whole URL would be replaced with
`#ZgotmplZ`.  This is to "defang" any potential attack in the URL,
rendering it useless.

Example: Given a site-wide `config.toml` that contains this menu entry:

    [[menu.main]]
        name = "IRC: #golang at freenode"
        url = "irc://irc.freenode.net/#golang"

The following template:

    <ul class="sidebar-menu">
      {{ range .Site.Menus.main }}
      <li><a href="{{ .URL }}">{{ .Name }}</a></li>
      {{ end }}
    </ul>

would produce `<li><a href="#ZgotmplZ">IRC: #golang at freenode</a></li>`
for the `irc://…` URL.

To fix this, add ` | safeURL` after `.URL` on the 3rd line, like this:

      <li><a href="{{ .URL | safeURL }}">{{ .Name }}</a></li>

With this change, we finally get `<li><a href="irc://irc.freenode.net/#golang">IRC: #golang at freenode</a></li>`
as intended.


### urlize
Takes a string and sanitizes it for usage in URLs, converts spaces to "-".

e.g. `<a href="/tags/{{ . | urlize }}">{{ . }}</a>`


### querify

Takes a set of key-value pairs and returns a [query string](https://en.wikipedia.org/wiki/Query_string) that can be appended to a URL. E.g.

    <a href="https://www.google.com?{{ (querify "q" "test" "page" 3) | safeURL }}">Search</a>

will be rendered as

    <a href="https://www.google.com?page=3&q=test">Search</a>


## Content Views

### Render
Takes a view to render the content with.  The view is an alternate layout, and should be a file name that points to a template in one of the locations specified in the documentation for [Content Views](/templates/views).

This function is only available on a piece of content, and in list context.

This example could render a piece of content using the content view located at `/layouts/_default/summary.html`:

    {{ range .Data.Pages }}
        {{ .Render "summary"}}
    {{ end }}



## Advanced

### apply

Given a map, array, or slice, returns a new slice with a function applied over it. Expects at least three parameters, depending on the function being applied. The first parameter is the sequence to operate on; the second is the name of the function as a string, which must be in the Hugo function map (generally, it is these functions documented here). After that, the parameters to the applied function are provided, with the string `"."` standing in for each element of the sequence the function is to be applied against. An example is in order:

    +++
    names: [ "Derek Perkins", "Joe Bergevin", "Tanner Linsley" ]
    +++

    {{ apply .Params.names "urlize" "." }} → [ "derek-perkins", "joe-bergevin", "tanner-linsley" ]

This is roughly equivalent to:

    {{ range .Params.names }}{{ . | urlize }}{{ end }}

However, it isn’t possible to provide the output of a range to the `delimit` function, so you need to `apply` it. A more complete example should explain this. Let's say you have two partials for displaying tag links in a post,  "post/tag/list.html" and "post/tag/link.html", as shown below.

    <!-- post/tag/list.html -->
    {{ with .Params.tags }}
    <div class="tags-list">
      Tags:
      {{ $len := len . }}
      {{ if eq $len 1 }}
        {{ partial "post/tag/link" (index . 0) }}
      {{ else }}
        {{ $last := sub $len 1 }}
        {{ range first $last . }}
          {{ partial "post/tag/link" . }},
        {{ end }}
        {{ partial "post/tag/link" (index . $last) }}
      {{ end }}
    </div>
    {{ end }}


    <!-- post/tag/link.html -->
    <a class="post-tag post-tag-{{ . | urlize }}" href="/tags/{{ . | urlize }}">{{ . }}</a>

This works, but the complexity of "post/tag/list.html" is fairly high; the Hugo template needs to perform special behaviour for the case where there’s only one tag, and it has to treat the last tag as special. Additionally, the tag list will be rendered something like "Tags: tag1 , tag2 , tag3" because of the way that the HTML is generated and it is interpreted by a browser.

This is Hugo. We have a better way. If this were your "post/tag/list.html" instead, all of those problems are fixed automatically (this first version separates all of the operations for ease of reading; the combined version will be shown after the explanation).

    <!-- post/tag/list.html -->
    {{ with .Params.tags }}
    <div class="tags-list">
      Tags:
      {{ $sort := sort . }}
      {{ $links := apply $sort "partial" "post/tag/link" "." }}
      {{ $clean := apply $links "chomp" "." }}
      {{ delimit $clean ", " }}
    </div>
    {{ end }}

In this version, we are now sorting the tags, converting them to links with "post/tag/link.html", cleaning off stray newlines, and joining them together in a delimited list for presentation. That can also be written as:

    <!-- post/tag/list.html -->
    {{ with .Params.tags }}
    <div class="tags-list">
      Tags:
      {{ delimit (apply (apply (sort .) "partial" "post/tag/link" ".") "chomp" ".") ", " }}
    </div>
    {{ end }}

`apply` does not work when receiving the sequence as an argument through a pipeline.

***

### base64Encode and base64Decode

`base64Encode` and `base64Decode` let you easily decode content with a base64 encoding and vice versa through pipes. Let's take a look at an example:


    {{ "Hello world" | base64Encode }}
    <!-- will output "SGVsbG8gd29ybGQ=" and -->

    {{ "SGVsbG8gd29ybGQ=" | base64Decode }}
    <!-- becomes "Hello world" again. -->

You can also pass other datatypes as argument to the template function which tries
to convert them. Now we use an integer instead of a string:


    {{ 42 | base64Encode | base64Decode }}
    <!-- will output "42". Both functions always return a string. -->

**Tip:** Using base64 to decode and encode becomes really powerful if we have to handle
responses of APIs.

    {{ $resp := getJSON "https://api.github.com/repos/spf13/hugo/readme"  }}
    {{ $resp.content | base64Decode | markdownify }}

The response of the GitHub API contains the base64-encoded version of the [README.md](https://github.com/spf13/hugo/blob/master/README.md) in the Hugo repository. Now we can decode it and parse the Markdown. The final output will look similar to the rendered version on GitHub.

***

### partialCached

See [Template Partials]({{< relref "templates/partials.md#cached-partials" >}}) for an explanation of the `partialCached` template function.


## .Site.GetPage
Every `Page` has a `Kind` attribute that shows what kind of page it is. While this attribute can be used to list pages of a certain `kind` using `where`, often it can be useful to fetch a single page by its path.

`GetPage` looks up an index page of a given `Kind` and `path`. This method may support regular pages in the future, but currently it is a convenient way of getting the index pages, such as the home page or a section, from a template:

    {{ with .Site.GetPage "section" "blog" }}{{ .Title }}{{ end }}

This method wil return `nil` when no page could be found, so the above will not print anything if the blog section isn't found.

The valid page kinds are: *home, section, taxonomy and taxonomyTerm.*

---
aliases:
- /doc/datafiles/
lastmod: 2015-08-04
date: 2015-01-22
menu:
  main:
    parent: extras
next: /extras/datadrivencontent
prev: /extras/robots-txt
title: Data Files
---

In addition to the [built-in variables](/templates/variables/) available from Hugo, you can specify your own custom data that can be accessed via templates or shortcodes.

Hugo supports loading data from [YAML](http://yaml.org/), [JSON](http://www.json.org/), and [TOML](https://github.com/toml-lang/toml) files located in the `data` directory.

**It even works with [LiveReload](/extras/livereload/).**

Data Files can also be used in [themes](/themes/overview/), but note: If the same `key` is used in both the main data folder and in the theme's data folder, the main one will win. So, for theme authors,  for theme specific data items that shouldn't be overridden, it can be wise to prefix the folder structure with a namespace, e.g. `mytheme/data/mytheme/somekey/...`. To check if any such duplicate exists, run hugo with the `-v` flag, e.g. `hugo -v`.

## The Data Folder

The `data` folder is where you can store additional data for Hugo to use when generating your site. Data files aren't used to generate standalone pages - rather they're meant to supplement the content files. This feature can extend the content in case your frontmatter would grow immensely. Or perhaps you want to show a larger dataset in a template (see example below). In both cases it's a good idea to outsource the data in their own file.  

These files must be YAML, JSON or TOML files (using either the `.yml`, `.yaml`, `.json` or `toml` extension) and the data will be accessible as a `map` in `.Site.Data`.

**The keys in this map will be a dot chained set of _path_, _filename_ and _key_ in file (if applicable).**

This is best explained with an example:

## Example: Jaco Pastorius' Solo Discography

[Jaco Pastorius](http://en.wikipedia.org/wiki/Jaco_Pastorius_discography) was a great bass player, but his solo discography is short enough to use as an example. [John Patitucci](http://en.wikipedia.org/wiki/John_Patitucci) is another bass giant.

The example below is a bit constructed, but it illustrates the flexibility of Data Files. It uses TOML as file format.

Given the files:

* `data/jazz/bass/jacopastorius.toml`
* `data/jazz/bass/johnpatitucci.toml`

`jacopastorius.toml` contains the content below, `johnpatitucci.toml` contains a similar list:

```
discography = [
"1974 – Modern American Music … Period! The Criteria Sessions",
"1974 – Jaco",
"1976 - Jaco Pastorius",
"1981 - Word of Mouth",
"1981 - The Birthday Concert (released in 1995)",
"1982 - Twins I & II (released in 1999)",
"1983 - Invitation",
"1986 - Broadway Blues (released in 1998)",
"1986 - Honestly Solo Live (released in 1990)",
"1986 - Live In Italy (released in 1991)",
"1986 - Heavy'n Jazz (released in 1992)",
"1991 - Live In New York City, Volumes 1-7.",
"1999 - Rare Collection (compilation)",
"2003 - Punk Jazz: The Jaco Pastorius Anthology (compilation)",
"2007 - The Essential Jaco Pastorius (compilation)"
]
```

The list of bass players can be accessed via `.Site.Data.jazz.bass`, a single bass player by adding the filename without the suffix, e.g. `.Site.Data.jazz.bass.jacopastorius`.

You can now render the list of recordings for all the bass players in a template:

```
{{ range $.Site.Data.jazz.bass }}
   {{ partial "artist.html" . }}
{{ end }}
```

And then in `partial/artist.html`:

```
<ul>
{{ range .discography }}
  <li>{{ . }}</li>
{{ end }}
</ul>
```

Discover a new favourite bass player? Just add another TOML-file.

## Example: Accessing named values in a Data File

Assuming you have the following YAML structure to your `User0123.yml` Data File located directly in `data/`

```
Name: User0123
"Short Description": "He is a **jolly good** fellow."
Achievements:
  - "Can create a Key, Value list from Data File"
  - "Learns Hugo"
  - "Reads documentation"
```

To render the `Short Description` in your `layout` File following code is required.

```
<div>Short Description of {{.Site.Data.User0123.Name}}: <p>{{ index .Site.Data.User0123 "Short Description" | markdownify }}</p></div>
```

Note the use of the `markdownify` template function. This will send the description through the Blackfriday Markdown rendering engine.

---
aliases:
- /doc/urls/
lastmod: 2016-05-07
date: 2014-01-03
menu:
  main:
    parent: extras
next: /community/mailing-list
notoc: true
prev: /extras/localfiles
title: URLs
---

## Pretty URLs

By default, Hugo creates content with 'pretty' URLs. For example,
content created at `/content/extras/urls.md` will be rendered at
`/public/extras/urls/index.html`, thus accessible from the browser
at http://example.com/extras/urls/.  No non-standard server-side
configuration is required for these pretty URLs to work.

If you would like to have what we call "ugly URLs",
e.g.&nbsp;http://example.com/extras/urls.html, you are in luck.
Hugo supports the ability to create your entire site with ugly URLs.
Simply add `uglyurls = true` to your site-wide `config.toml`,
or use the `--uglyURLs=true` flag on the command line.

If you want a specific piece of content to have an exact URL, you can
specify this in the front matter under the `url` key. See [Content
Organization](/content/organization/) for more details.

## Canonicalization

By default, all relative URLs encountered in the input are left unmodified,
e.g. `/css/foo.css` would stay as `/css/foo.css`,
i.e. `canonifyURLs` defaults to `false`.

By setting `canonifyURLs` to `true`, all relative URLs would instead
be *canonicalized* using `baseURL`.  For example, assuming you have
`baseURL = http://yoursite.example.com/` defined in the site-wide
`config.toml`, the relative URL `/css/foo.css` would be turned into
the absolute URL `http://yoursite.example.com/css/foo.css`.

Benefits of canonicalization include fixing all URLs to be absolute, which may
aid with some parsing tasks.  Note though that all real browsers handle this
client-side without issues.

Benefits of non-canonicalization include being able to have resource inclusion
be scheme-relative, so that http vs https can be decided based on how this
page was retrieved.

> Note: In the May 2014 release of Hugo v0.11, the default value of `canonifyURLs` was switched from `true` to `false`, which we think is the better default and should continue to be the case going forward. So, please verify and adjust your website accordingly if you are upgrading from v0.10 or older versions.

To find out the current value of `canonifyURLs` for your website, you may use the handy `hugo config` command added in v0.13:

    hugo config | grep -i canon

Or, if you are on Windows and do not have `grep` installed:

    hugo config | FINDSTR /I canon

## Relative URLs

By default, all relative URLs are left unchanged by Hugo,
which can be problematic when you want to make your site browsable from a local file system.

Setting `relativeURLs` to `true` in the site configuration will cause Hugo to rewrite all relative URLs to be relative to the current content.

For example, if the `/post/first/` page contained a link with a relative URL of `/about/`, Hugo would rewrite that URL to `../../about/`.

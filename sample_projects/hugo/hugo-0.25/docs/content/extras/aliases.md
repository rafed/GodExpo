---
aliases:
- /doc/redirects/
- /doc/alias/
- /doc/aliases/
lastmod: 2015-12-23
date: 2013-07-09
menu:
  main:
    parent: extras
next: /extras/analytics
prev: /taxonomies/methods
title: Aliases
---

For people migrating existing published content to Hugo, there's a good chance you need a mechanism to handle redirecting old URLs.

Luckily, redirects can be handled easily with _aliases_ in Hugo.

## Example

Given a post on your current Hugo site, with a path of:

``content/posts/my-awesome-blog-post.md``

... you create an "aliases" section in the frontmatter of your post, and add previous paths to that.

### TOML frontmatter

```toml
+++
        ...
aliases = [
    "/posts/my-original-url/",
    "/2010/01/01/even-earlier-url.html"
]
        ...
+++
```

### YAML frontmatter

```yaml
---
        ...
aliases:
    - /posts/my-original-url/
    - /2010/01/01/even-earlier-url.html
        ...
---
```

Now when you visit any of the locations specified in aliases, _assuming the same site domain_, you'll be redirected to the page they are specified on.

## Important Behaviors

1. *Hugo makes no assumptions about aliases. They also don't change based
on your UglyURLs setting. You need to provide absolute path to your webroot
and the complete filename or directory.*

2. *Aliases are rendered prior to any content and will be overwritten by
any content with the same location.*

## Multilingual example

On [multilingual sites]({{< relref "content/multilingual.md" >}}), each translation of a post can have unique aliases. To use the same alias across multiple languages, prefix it with the language code.

In `/posts/my-new-post.es.md`:

```yaml
---
aliases:
    - /es/posts/my-original-post/
---
```

## How Hugo Aliases Work

When aliases are specified, Hugo creates a physical folder structure to match the alias entry, and, an html file specifying the canonical URL for the page, and a redirect target.

Assuming a baseURL of `mysite.tld`, the contents of the html file will look something like:

```html
<!DOCTYPE html>
<html>
  <head>
    <title>http://mysite.tld/posts/my-original-url</title>
    <link rel="canonical" href="http://mysite.tld/posts/my-original-url"/>
    <meta http-equiv="content-type" content="text/html; charset=utf-8"/>
    <meta http-equiv="refresh" content="0; url=http://mysite.tld/posts/my-original-url"/>
  </head>
</html>
```

The `http-equiv="refresh"` line is what performs the redirect, in 0 seconds in this case.

## Customizing

You may customize this alias page by creating an alias.html template in the
layouts folder of your site.  In this case, the data passed to the template is

* Permalink - the link to the page being aliased
* Page - the Page data for the page being aliased

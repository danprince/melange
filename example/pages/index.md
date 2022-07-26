---
title: My Site
---

This is an example index page that lists out all the posts in this directory.

{{ range pages }}
- [{{ .Data.title }}]({{ .Url }})
{{end}}

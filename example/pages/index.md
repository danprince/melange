---
title: The Title
date: 2022-07-26
---

Some content. Don't forget it's _markdown_.

{{ "./_counter.tsx" | render }}
{{ "./_counter.tsx" | hydrate }}
{{ "./_counter.tsx" | dynamic }}

{{ range pages }}
- [{{ .Data.title }}]({{ .Url }})
{{end}}

---
title: Hello World
---

Now _this_ is __markdown__.

I only render at the server
{{ render "./_counter.tsx" "count" 1 }}

I render at the server, then again at the client
{{ render "./_counter.tsx" "count" 2 | client_load }}

I only render at the client
{{ render "./_counter.tsx" "count" 3 | client_only }}

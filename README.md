# Melange

- Files ending with .md become .html
- Every file is templated into _theme.html if it exists, if not use the default theme

- Files can render Preact components in 3 ways
  1. Static render `{{ "counter.tsx" | render }}`
  2. Hydrate `{{ "counter.tsx" | hydrate }}`
  3. Dynamic render `{{ "counter.tsx" | dynamic }}`

Initially these functions will replace the content with a marker token, that allows us to swap the value out for the HTML we get from actually rendering the component asynchronously later. These functions will wrap that marker token in a div with an ID that allows the component to be "rehydrated" at the client side, if necessary.

Once all pages have been rendered, esbuild will produce multiple bundles from the rendered components.
- A static nodejs bundle that will render the static versions of the elements for each page. 
- A browser bundle for each page that will hydrate the appropriate elements at runtime.

Go has to ask a Nodejs process to evaluate the static bundle, then the response is used to replace the marker tokens in the evaluated page templates.

Finally the appropriate scripts/styles are injected into the pages and the everything is copied/written to disk.

## TODO
- [x] Use long-running node process to prevent paying for once-per-build startup
  - [x] Don't use stdio (prevent console.log from messing with output)
- [x] Support custom _theme.html files
- [x] Fix collisions between hydrations IDs across separate files
- [x] Fix rendering order to make `{{ pages }}` deterministic (render index.md last)
- [x] Figure out how to make hydration generic (preact/react swap)
- [x] Support build time props on components
- [ ] Try npm free react-cdn/preact-cdn frameworks?
- [ ] Support a sensible set of assets
  - Useful starting place https://github.com/remix-run/remix/blob/37490ad24dee2af81f5c309ff0fa0e6e84f965bd/packages/remix-dev/compiler/loaders.ts
- [ ] esbuild plugin that strips non-js files from the server build?
- [ ] Syntax highlighting
- [ ] Bundle function that adds a script without hydrations
- [ ] Preflight checks for dependencies
  - [ ] Node
  - [ ] Frameworks
- [ ] Make common error presentation as friendly as possible
  - [ ] Esbuild bundler errors (probably parse related)
  - [ ] Node execution errors
  - [ ] Parse time errors inside templates
  - [ ] Runtime errors inside templates
- [ ] Tests
  - [ ] Ignored pages aren't copied/built
  - [ ] Site is rendered from leaf to root
  - [ ] Frontmatter is parsed
  - [ ] Test that page with no elements is has no script tags
  - [ ] Test that page with static elements has no script tags

## Go vs Node (Bun/Deno)
+ Faster
+ Static binary
+ Built in templates
+ Native esbuild
+ Simpler compilations (than TS)
- Small markdown ecosystem
- No MDX option
- Relies on node for execution
- Few/plugins for native esbuild

## Known
- [ ] Make jsxImportSource work (might be blocked by https://github.com/evanw/esbuild/pull/2349)
- [x] Support prod builds
- [ ] Solve the problem of SSR components not generating CSS for client

## Tests

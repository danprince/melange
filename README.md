# Melange

- Files starting with _ or . are ignored
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
- Use long-running node process to prevent paying for once-per-build startup

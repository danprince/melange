package main

import (
	"fmt"
	"path"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

type framework struct {
	esbuildOptions api.BuildOptions
	staticExternal []string
	staticBundle   func(config *config) string
	clientBundle   func(page *page) string
}

var preact = framework{
	staticExternal: []string{"preact", "preact-render-to-string"},
	esbuildOptions: api.BuildOptions{},
	staticBundle: func(config *config) string {
		var builder strings.Builder

		builder.WriteString("import { h } from \"preact\";\n")
		builder.WriteString("import { render } from \"preact-render-to-string\";\n")
		builder.WriteString("let elements = {};")

		for _, page := range config.pages {
			for _, element := range page.elements {
				if element.ssr {
					builder.WriteString(fmt.Sprintf(
						"import { default as C%s } from \"%s\";\n",
						element.id,
						path.Join(page.dir, element.src),
					))
					builder.WriteString(fmt.Sprintf(
						"elements.%s = render(h(C%s, null));\n",
						element.id,
						element.id,
					))
				}
			}
		}

		builder.WriteString("process.stdout.write(JSON.stringify(elements))")
		return builder.String()
	},
	clientBundle: func(page *page) string {
		var builder strings.Builder
		builder.WriteString("import { h, hydrate, Fragment } from \"preact\";\n")

		for _, element := range page.elements {
			if element.csr {
				builder.WriteString(fmt.Sprintf(
					"import { default as %s } from \"%s\";\n",
					element.id,
					element.src,
				))
				builder.WriteString(fmt.Sprintf(
					"hydrate(h(%s, %s), document.getElementById(\"%s\"));\n",
					element.id,
					toJson(&element.props),
					element.id,
				))
			}
		}

		return builder.String()
	},
}

var react = framework{
	staticExternal: []string{"react", "react-dom"},
	esbuildOptions: api.BuildOptions{},
	staticBundle: func(config *config) string {
		var builder strings.Builder

		builder.WriteString("import * as React from \"react\";\n")
		builder.WriteString("import { renderToString } from \"react-dom/server\";\n")
		builder.WriteString("let elements = {};\n")

		for _, page := range config.pages {
			for _, element := range page.elements {
				builder.WriteString(fmt.Sprintf(
					"import C%s from \"%s\";\n",
					element.id,
					path.Join(page.dir, element.src),
				))
				builder.WriteString(fmt.Sprintf(
					"elements.%s = renderToString(React.createElement(C%s));\n",
					element.id,
					element.id,
				))
			}
		}

		builder.WriteString("process.stdout.write(JSON.stringify(elements))")
		return builder.String()
	},
	clientBundle: func(page *page) string {
		var builder strings.Builder
		builder.WriteString("import * as React from \"react\";\n")
		builder.WriteString("import { hydrateRoot } from \"react-dom/client\";\n")

		for _, element := range page.elements {
			if element.csr {
				builder.WriteString(fmt.Sprintf(
					"import { default as %s } from \"%s\";\n",
					element.id,
					element.src,
				))
				builder.WriteString(fmt.Sprintf(
					"hydrateRoot(document.getElementById(\"%s\"), React.createElement(%s, %s));\n",
					element.id,
					element.id,
					toJson(&element.props),
				))
			}
		}

		return builder.String()
	},
}

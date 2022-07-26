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
	esbuildOptions: api.BuildOptions{
		JSXMode:     api.JSXModeTransform,
		JSXFactory:  "h",
		JSXFragment: "Fragment",
	},
	staticBundle: func(config *config) string {
		var builder strings.Builder

		builder.WriteString("import { h } from \"preact\";\n")
		builder.WriteString("import { render } from \"preact-render-to-string\";\n")
		builder.WriteString("let elements = {};")

		for _, page := range config.pages {
			for _, element := range page.elements {
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

		builder.WriteString("process.stdout.write(JSON.stringify(elements))")
		return builder.String()
	},
	clientBundle: func(page *page) string {
		var builder strings.Builder
		builder.WriteString("import { h, hydrate, Fragment } from \"preact\";\n")

		for _, element := range page.elements {
			if element.hydrate {
				builder.WriteString(fmt.Sprintf(
					"import { default as %s } from \"%s\";\n",
					element.id,
					element.src,
				))
				builder.WriteString(fmt.Sprintf(
					"hydrate(h(%s), document.getElementById(\"%s\"));\n",
					element.id,
					element.id,
				))
			}
		}

		return builder.String()
	},
}

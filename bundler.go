package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

func bundle(config *config) error {
	if err := createStaticBundle(config); err != nil {
		return err
	}

	if err := createClientBundles(config); err != nil {
		return err
	}

	return nil
}

func createStaticBundle(config *config) error {
	var builder strings.Builder

	builder.WriteString("import { h } from \"preact\";\n")
	builder.WriteString("import { render } from \"preact-render-to-string\";\n")
	builder.WriteString("let elements = {};")

	for _, page := range config.pages {
		for _, hydration := range page.hydrations {
			builder.WriteString(fmt.Sprintf(
				"import { default as C%s } from \"%s\";\n",
				hydration.elementId,
				path.Join(page.dir, hydration.src),
			))
			builder.WriteString(fmt.Sprintf(
				"elements.%s = render(h(C%s, null));\n",
				hydration.elementId,
				hydration.elementId,
			))
		}
	}

	builder.WriteString("process.stdout.write(JSON.stringify(elements))")

	outfile := path.Join(config.cacheDir, "static-bundle.js")

	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   builder.String(),
			ResolveDir: config.inputDir,
			Sourcefile: "static-bundle.js",
			Loader:     api.LoaderJS,
		},
		Write:       true,
		Bundle:      true,
		Metafile:    true,
		Sourcemap:   api.SourceMapExternal,
		Outfile:     outfile,
		Platform:    api.PlatformNode,
		Format:      api.FormatCommonJS,
		JSXMode:     api.JSXModeTransform,
		JSXFactory:  "h",
		JSXFragment: "Fragment",
		External:    []string{"preact", "preact-render-to-string"},
	})

	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			log.Println(err)
		}

		return errors.New("bundler failed")
	}

	// TODO: This is a huge bottleneck (waiting for node to start)
	var hydrations map[string]string
	hydrationJson, err := exec.Command("node", outfile).Output()

	if err != nil {
		return err
	}

	err = json.Unmarshal((hydrationJson), &hydrations)

	if err != nil {
		return err
	}

	for _, page := range config.pages {
		for _, hydration := range page.hydrations {
			html := hydrations[hydration.elementId]
			page.Contents = strings.Replace(page.Contents, hydration.token, html, -1)
		}
	}

	return nil
}

func createClientBundles(config *config) error {
	var entryPoints []string

	for _, page := range config.pages {
		if len(page.hydrations) > 0 {
			entryPoint := fmt.Sprintf("page:%s", page.id)
			entryPoints = append(entryPoints, entryPoint)
		}
	}

	result := api.Build(api.BuildOptions{
		EntryPoints: entryPoints,
		EntryNames:  "[name]",
		Outdir:      path.Join(config.outputDir, "assets"),
		Write:       true,
		Bundle:      true,
		Metafile:    true,
		Sourcemap:   api.SourceMapExternal,
		//MinifyWhitespace: true,
		//MinifyIdentifiers: true,
		//MinifySyntax: true,
		Platform:    api.PlatformBrowser,
		Format:      api.FormatIIFE,
		JSXMode:     api.JSXModeTransform,
		JSXFactory:  "h",
		JSXFragment: "Fragment",
		Plugins:     []api.Plugin{hydratePagesPlugin(config)},
	})

	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			id := strings.ReplaceAll(err.Location.File, "page:", "")
			page := config.pages[id]
			fmt.Printf("%s in %s\n", err.Text, page.relativePath)
		}

		return errors.New("bundler failed")
	}

	for _, page := range config.pages {
		scripts := []string{}
		styles := []string{}

		for _, file := range result.OutputFiles {
			if strings.Contains(file.Path, page.id) {
				ext := path.Ext(file.Path)
				relpath := file.Path[len(config.outputDir):]
				switch ext {
				case ".js":
					scripts = append(scripts, relpath)
				case ".css":
					styles = append(styles, relpath)
				case ".map":
					// do nothing
				default:
					fmt.Printf("unrecognised outfile extension %s\n", ext)
				}
			}
		}

		if len(scripts) == 0 && len(styles) == 0 {
			continue
		}

		var inject strings.Builder

		for _, href := range styles {
			inject.WriteString(fmt.Sprintf(`<link rel="stylesheet" href="%s">`, href))
			inject.WriteByte('\n')
		}

		for _, src := range scripts {
			inject.WriteString(fmt.Sprintf(`<script defer src="%s"></script>`, src))
			inject.WriteByte('\n')
		}

		inject.WriteString("</head>")

		page.Contents = strings.Replace(page.Contents, "</head>", inject.String(), 1)
	}

	return nil
}

func hydratePagesPlugin(config *config) api.Plugin {
	filter := "page:"
	namespace := "page"

	return api.Plugin{
		Name: "hydrate-pages",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{
				Filter: filter,
			}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				return api.OnResolveResult{
					Path:      args.Path,
					Namespace: namespace,
				}, nil
			})

			build.OnLoad(api.OnLoadOptions{
				Filter:    filter,
				Namespace: namespace,
			}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				id := strings.Replace(args.Path, filter, "", 1)
				page := config.pages[id]

				var builder strings.Builder
				builder.WriteString("import { h, hydrate, Fragment } from \"preact\";\n")
				hydrations := 0

				for _, hydration := range page.hydrations {
					if hydration.hydrate {
						hydrations += 1
						builder.WriteString(fmt.Sprintf(
							"import { default as %s } from \"%s\";\n",
							hydration.elementId,
							hydration.src,
						))
						builder.WriteString(fmt.Sprintf(
							"hydrate(h(%s), document.getElementById(\"%s\"));\n",
							hydration.elementId,
							hydration.elementId,
						))
					}
				}

				var contents string

				if hydrations > 0 {
					contents = builder.String()
				}

				return api.OnLoadResult{
					Contents:   &contents,
					Loader:     api.LoaderJS,
					ResolveDir: path.Dir(page.path),
				}, nil
			})
		},
	}
}

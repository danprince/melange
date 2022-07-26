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
	contents := config.framework.staticBundle(config)
	outfile := path.Join(config.cacheDir, "static-bundle.js")

	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   contents,
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
		External:    config.framework.staticExternal,
		Incremental: !config.production,
	})

	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			log.Println(err)
		}

		return errors.New("bundler failed")
	}

	// TODO: This is a huge bottleneck (waiting for node to start)
	var renderedHtml map[string]string
	renderedHtmlJson, err := exec.Command("node", outfile).Output()

	if err != nil {
		return err
	}

	err = json.Unmarshal(renderedHtmlJson, &renderedHtml)

	if err != nil {
		return err
	}

	for _, page := range config.pages {
		for _, element := range page.elements {
			html := renderedHtml[element.id]
			page.Contents = strings.Replace(page.Contents, element.token, html, -1)
		}
	}

	return nil
}

func createClientBundles(config *config) error {
	var entryPoints []string

	for _, page := range config.pages {
		for _, element := range page.elements {
			if element.csr {
				entryPoint := fmt.Sprintf("page:%s", page.id)
				entryPoints = append(entryPoints, entryPoint)
				break
			}
		}
	}

	entryNames := "[name]"

	if config.production {
		entryNames = "[name]-[hash]"
	}

	result := api.Build(api.BuildOptions{
		EntryPoints:       entryPoints,
		EntryNames:        entryNames,
		Outdir:            path.Join(config.outputDir, "assets"),
		Write:             true,
		Bundle:            true,
		Metafile:          true,
		Sourcemap:         api.SourceMapExternal,
		MinifyWhitespace:  config.production,
		MinifyIdentifiers: config.production,
		MinifySyntax:      config.production,
		Incremental:       !config.production,
		Platform:          api.PlatformBrowser,
		Format:            api.FormatIIFE,
		Plugins:           []api.Plugin{hydratePagesPlugin(config)},
	})

	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			id := strings.ReplaceAll(err.Location.File, "page:", "")
			page := config.pages[id]
			fmt.Printf("%s in %s\n", err.Text, page.relPath)
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
				contents := config.framework.clientBundle(page)

				return api.OnLoadResult{
					Contents:   &contents,
					Loader:     api.LoaderJS,
					ResolveDir: path.Dir(page.absPath),
				}, nil
			})
		},
	}
}

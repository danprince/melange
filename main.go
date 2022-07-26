package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	html "github.com/yuin/goldmark/renderer/html"
)

var (
	//go:embed static/theme.gohtml
	themeHtml string

	//go:embed static/theme.css
	themeCss string
)

type asset struct {
	path         string
	relativePath string
}

type hydration struct {
	elementId string
	src       string
	hydrate   bool
	ssr       bool
	token     string
}

type page struct {
	id           string
	path         string
	dir          string
	relativePath string
	template     *template.Template
	Contents     string
	Url          string
	Data         map[string]any
	Name         string
	hydrations   []hydration
}

type config struct {
	inputDir    string
	outputDir   string
	pagesDir    string
	cacheDir    string
	pages       map[string]*page
	assets      []*asset
	directories []string
	markdown    goldmark.Markdown
	template    *template.Template
}

func createConfig(inputDir string) config {
	outputDir := path.Join(inputDir, "_site")
	pagesDir := path.Join(inputDir, "pages")
	cacheDir := path.Join(inputDir, "node_modules/.cache/melange")

	template, err := template.New("page").Parse(themeHtml)

	if err != nil {
		log.Fatal(err)
	}

	return config{
		inputDir:  inputDir,
		outputDir: outputDir,
		pagesDir:  pagesDir,
		cacheDir:  cacheDir,
		template:  template,
		markdown:  createMarkdownRenderer(),
		pages:     map[string]*page{},
	}
}

func createMarkdownRenderer() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
}

func shouldIgnore(name string) bool {
	return name[0] == '_' || name[0] == '.'
}

func shortHash(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum32())
}

func parseSite(config *config) {
	stack := []string{config.pagesDir}

	for len(stack) > 0 {
		n := len(stack)
		dir := stack[n-1]
		stack = stack[:n-1]
		entries, err := ioutil.ReadDir(dir)

		if err != nil {
			log.Fatal(err)
		}

		for _, entry := range entries {
			name := entry.Name()
			pathName := path.Join(dir, name)
			relativePath := pathName[len(config.pagesDir):]

			if shouldIgnore(name) {
				continue
			} else if entry.IsDir() {
				stack = append(stack, pathName)
				config.directories = append(config.directories, relativePath)
			} else if filepath.Ext(name) == ".md" {
				id := shortHash(pathName)
				url := relativePath
				url = strings.Replace(url, ".md", ".html", 1)
				url = strings.Replace(url, "index.html", "", 1)

				config.pages[id] = &page{
					id:           shortHash(pathName),
					path:         pathName,
					dir:          dir,
					relativePath: relativePath,
					Name:         name,
					Url:          url,
				}
			} else {
				config.assets = append(config.assets, &asset{
					path:         pathName,
					relativePath: relativePath,
				})
			}
		}
	}
}

func (page *page) addElement(src string, hydrate bool, ssr bool) string {
	elementId := fmt.Sprintf("$hydrate_%d", len(page.hydrations))
	token := fmt.Sprintf("<!-- %s -->", elementId)

	if hydrate {
		page.hydrations = append(page.hydrations, hydration{
			elementId: elementId,
			src:       src,
			hydrate:   hydrate,
			ssr:       ssr,
			token:     token,
		})
	}

	if ssr && !hydrate {
		return token
	} else {
		return fmt.Sprintf("<div id=\"%s\">%s</div>", elementId, token)
	}
}

func (config *config) getPageIndex(dir string) []*page {
	var index []*page

	for _, page := range config.pages {
		if (page.dir == dir && page.Name != "index.md") ||
			(path.Dir(page.dir) == dir && page.Name == "index.md") {
			index = append(index, page)
		}
	}

	return index
}

func readPage(p *page, config *config) {
	contents, err := os.ReadFile(p.path)

	if err != nil {
		log.Fatal(err)
	}

	templateFuncs := template.FuncMap{
		"render": func(entry string) string {
			return p.addElement(entry, false, true)
		},
		"hydrate": func(entry string) string {
			return p.addElement(entry, true, true)
		},
		"dynamic": func(entry string) string {
			return p.addElement(entry, true, false)
		},
		"pages": func() []*page {
			return config.getPageIndex(p.dir)
		},
	}

	tpl, err := template.New("page").Funcs(templateFuncs).Parse(string(contents))

	if err != nil {
		log.Fatal(err)
	}

	p.template = tpl
}

func readPages(config *config) {
	for _, page := range config.pages {
		readPage(page, config)
	}
}

type renderContext struct {
	Page          *page
	Config        *config
	DefaultStyles string
}

func renderPage(page *page, config *config) {
	scope := renderContext{Page: page, Config: config, DefaultStyles: themeCss}

	// It's a three step process to render a page.

	// 1. Execute the page's own template. This is a markdown template that will
	// handle any in-page templating.
	var pageBuf bytes.Buffer
	err := page.template.Execute(&pageBuf, scope)

	if err != nil {
		log.Fatal(err)
	}

	// 2. Convert the output from the previous step to HTML.
	var htmlbuf bytes.Buffer
	ctx := parser.NewContext()
	err = config.markdown.Convert(pageBuf.Bytes(), &htmlbuf, parser.WithContext(ctx))

	if err != nil {
		log.Fatal(err)
	}

	page.Data = meta.Get(ctx)
	page.Contents = htmlbuf.String()

	// 3. Execute the theme template to render the complete page, with layout.
	var buf bytes.Buffer
	err = config.template.Execute(&buf, scope)

	if err != nil {
		log.Fatal(err)
	}

	page.Contents = buf.String()
}

func renderPages(config *config) {
	for _, page := range config.pages {
		renderPage(page, config)
	}
}

func writeSite(config *config) {
	err := os.MkdirAll(config.outputDir, os.ModePerm)

	if err != nil {
		log.Fatal(err)
	}

	for _, dir := range config.directories {
		outdir := path.Join(config.outputDir, dir)
		err := os.MkdirAll(outdir, os.ModePerm)

		if err != nil {
			log.Fatal(err)
		}
	}

	for _, page := range config.pages {
		outputPath := path.Join(config.outputDir, page.relativePath)
		outputPath = strings.Replace(outputPath, ".md", ".html", 1)
		f, err := os.Create(outputPath)

		if err != nil {
			log.Fatal(err)
		}

		defer f.Close()

		_, err = f.WriteString(page.Contents)

		if err != nil {
			log.Fatal(err)
		}
	}

	for _, asset := range config.assets {
		outputPath := path.Join(config.outputDir, asset.relativePath)

		src, err := os.Open(asset.path)

		if err != nil {
			log.Fatal(err)
		}

		dst, err := os.Create(outputPath)

		if err != nil {
			log.Fatal(err)
		}

		_, err = io.Copy(dst, src)

		if err != nil {
			log.Fatal(err)
		}
	}
}

func build(dir string) (*config, error) {
	config := createConfig(dir)
	parseSite(&config)
	readPages(&config)
	renderPages(&config)
	if err := bundle(&config); err != nil {
		return nil, err
	}
	writeSite(&config)
	return &config, nil
}

func timedBuild(dir string) (*config, error) {
	start := time.Now()
	config, err := build(dir)
	fmt.Printf("built site in %s\n", time.Since(start))
	return config, err
}

func serve(dir string) {
	config, err := build(dir)

	if err != nil {
		log.Fatal(err)
	}

	fs := http.FileServer(http.Dir(config.outputDir))

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config, err = timedBuild(dir)

		if err != nil {
			http.Error(w, "Build failed", 500)
			log.Fatal(err)
		}

		fs.ServeHTTP(w, r)
	}))

	err = http.ListenAndServe(":8000", nil)

	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	cwdFlag := flag.String("cwd", "", "")
	flag.Parse()

	if cwdFlag != nil {
		err := os.Chdir(*cwdFlag)
		if err != nil {
			log.Fatal(err)
		}
	}

	inputDir, _ := os.Getwd()
	//timedBuild(inputDir)
	serve(inputDir)
}

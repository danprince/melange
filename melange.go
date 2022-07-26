package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
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
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	html "github.com/yuin/goldmark/renderer/html"
)

var (
	//go:embed theme.gohtml
	themeHtml string

	//go:embed theme.css
	themeCss string
)

type asset struct {
	absPath string
	relPath string
}

type props map[string]any

type element struct {
	id    string
	src   string
	csr   bool
	ssr   bool
	token string
	props props
}

type page struct {
	id       string
	absPath  string
	dir      string
	relPath  string
	depth    int
	template *template.Template
	Contents string
	Url      string
	Data     map[string]any
	Name     string
	elements []*element
}

type config struct {
	production  bool
	inputDir    string
	outputDir   string
	pagesDir    string
	cacheDir    string
	pages       map[string]*page
	assets      []*asset
	directories []string
	markdown    goldmark.Markdown
	template    *template.Template
	framework   framework
}

func createConfig(inputDir string, production bool) config {
	outputDir := path.Join(inputDir, "_site")
	pagesDir := path.Join(inputDir, "pages")
	cacheDir := path.Join(inputDir, "node_modules/.cache/melange")

	template, err := template.New("page").Parse(themeHtml)

	if err != nil {
		log.Fatal(err)
	}

	return config{
		production: production,
		inputDir:   inputDir,
		outputDir:  outputDir,
		pagesDir:   pagesDir,
		cacheDir:   cacheDir,
		template:   template,
		markdown:   createMarkdownRenderer(),
		pages:      map[string]*page{},
		framework:  preact,
	}
}

func createMarkdownRenderer() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
			extension.GFM,
			extension.Footnote,
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
}

func shouldIgnore(name string) bool {
	return name[0] == '_' || name[0] == '.'
}

func isPageFile(name string) bool {
	return filepath.Ext(name) == ".md"
}

func shortHash(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum32())
}

type crawldir struct {
	name  string
	depth int
}

func crawlSite(config *config) {
	stack := []crawldir{{config.pagesDir, 0}}

	for len(stack) > 0 {
		end := len(stack) - 1
		dir := stack[end]
		stack = stack[:end]
		entries, err := ioutil.ReadDir(dir.name)

		if err != nil {
			log.Fatal(err)
		}

		for _, entry := range entries {
			name := entry.Name()
			absPath := path.Join(dir.name, name)
			relPath := absPath[len(config.pagesDir):]

			if shouldIgnore(name) {
				continue
			} else if entry.IsDir() {
				stack = append(stack, crawldir{name: absPath, depth: dir.depth + 1})
				config.directories = append(config.directories, relPath)
			} else if isPageFile(absPath) {
				id := shortHash(absPath)
				depth := dir.depth

				// index files are treated as part of the parent directory
				if name == "index.md" {
					depth -= 1
				}

				config.pages[id] = &page{
					id:      id,
					dir:     dir.name,
					depth:   depth,
					absPath: absPath,
					relPath: relPath,
					Name:    name,
					Url:     relPath,
				}
			} else {
				config.assets = append(config.assets, &asset{
					absPath: absPath,
					relPath: relPath,
				})
			}
		}
	}
}

func toJson(props *props) string {
	out, _ := json.Marshal(props)
	return string(out)
}

func (page *page) addElement(src string, props props) *element {
	hash := shortHash(fmt.Sprintf("%s%d", page.relPath, len(page.elements)))
	id := fmt.Sprintf("$hydrate_%s", hash)
	token := fmt.Sprintf("<!-- %s -->", id)

	el := element{
		id:    id,
		src:   src,
		ssr:   true,
		csr:   false,
		token: token,
		props: props,
	}

	page.elements = append(page.elements, &el)
	return &el
}

func (e *element) String() string {
	if e.ssr && !e.csr {
		return e.token
	} else {
		return fmt.Sprintf("<div id=\"%s\">%s</div>", e.id, e.token)
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

func parseProps(kvs ...any) props {
	props := map[string]any{}
	for i := 0; i < len(kvs); i += 2 {
		props[kvs[i].(string)] = kvs[i+1]
	}
	return props
}

func readPage(p *page, config *config) error {
	contents, err := os.ReadFile(p.absPath)

	if err != nil {
		log.Fatal(err)
	}

	templateFuncs := template.FuncMap{
		"render": func(entry string, args ...any) *element {
			props := parseProps(args...)
			el := p.addElement(entry, props)
			return el
		},
		"client_load": func(element *element) *element {
			element.csr = true
			element.ssr = true
			return element
		},
		"client_only": func(element *element) *element {
			element.csr = true
			element.ssr = false
			return element
		},
		"pages": func() []*page {
			return config.getPageIndex(p.dir)
		},
	}

	tpl, err := template.New("page").Funcs(templateFuncs).Parse(string(contents))

	if err != nil {
		return err
	}

	p.template = tpl
	return nil
}

func readPages(config *config) error {
	for _, page := range config.pages {
		if err := readPage(page, config); err != nil {
			return err
		}
	}

	return nil
}

type renderContext struct {
	Page          *page
	Config        *config
	DefaultStyles string
}

func renderPage(page *page, config *config) error {
	scope := renderContext{Page: page, Config: config, DefaultStyles: themeCss}

	// 1. Execute the page's own template. This is a markdown template that will
	// handle any in-page templating.
	var pageBuf bytes.Buffer
	err := page.template.Execute(&pageBuf, scope)

	if err != nil {
		return err
	}

	// 2. Convert the output from the previous step to HTML.
	var htmlbuf bytes.Buffer
	ctx := parser.NewContext()
	err = config.markdown.Convert(pageBuf.Bytes(), &htmlbuf, parser.WithContext(ctx))

	if err != nil {
		return err
	}

	page.Url = strings.Replace(page.Url, ".md", ".html", 1)
	page.Url = strings.Replace(page.Url, "index.html", "", 1)
	page.Data = meta.Get(ctx)
	page.Contents = htmlbuf.String()

	// 3. Execute the theme template to render the complete page, with layout.
	var buf bytes.Buffer
	err = config.template.Execute(&buf, scope)

	if err != nil {
		return err
	}

	page.Contents = buf.String()
	return nil
}

func renderPages(config *config) error {
	pages := make([]*page, 0, len(config.pages))

	for _, page := range config.pages {
		pages = append(pages, page)
	}

	sort.SliceStable(pages, func(i, j int) bool {
		return pages[i].depth > pages[j].depth
	})

	for _, page := range pages {
		if err := renderPage(page, config); err != nil {
			return err
		}
	}

	return nil
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
		outputPath := path.Join(config.outputDir, page.relPath)
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
		outputPath := path.Join(config.outputDir, asset.relPath)

		src, err := os.Open(asset.absPath)

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

func Build(dir string, prod bool) (*config, error) {
	start := time.Now()
	config := createConfig(dir, prod)
	crawlSite(&config)

	if err := readPages(&config); err != nil {
		return nil, err
	}

	if err := renderPages(&config); err != nil {
		return nil, err
	}

	if err := bundle(&config); err != nil {
		return nil, err
	}

	writeSite(&config)
	fmt.Printf("built site in %s\n", time.Since(start))
	return &config, nil
}

func Serve(dir string, prod bool) {
	config, err := Build(dir, prod)

	if err != nil {
		log.Fatal(err)
	}

	fs := http.FileServer(http.Dir(config.outputDir))

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rebuild the site whenever an HTML file is requested
		if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, ".html") {
			config, err = Build(dir, prod)
		}

		if err != nil {
			http.Error(w, "Build failed", 500)
			log.Println(err)
		}

		fs.ServeHTTP(w, r)
	}))

	fmt.Println("serving site at http://localhost:8000...")
	err = http.ListenAndServe(":8000", nil)

	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	var cwd string
	var serve bool

	flag.BoolVar(&serve, "serve", false, "serve the site and rebuild for each request")
	flag.StringVar(&cwd, "cwd", "", "cwd of your site")
	flag.Parse()

	if cwd != "" {
		err := os.Chdir(cwd)
		if err != nil {
			log.Fatal(err)
		}
	}

	inputDir, _ := os.Getwd()

	if serve {
		Serve(inputDir, false)
	} else {
		Build(inputDir, true)
	}
}

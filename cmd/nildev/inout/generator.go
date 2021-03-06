package inout

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"github.com/nildev/lib/codegen"
	"github.com/nildev/lib/log"
	"github.com/nildev/tools/cmd/nildev/template"
)

type (
	defaultGenerator struct {
		tpl        string
		outputFile string
		vm         *viewModel
	}

	viewModel struct {
		PackageName string
		BasePattern string
		RoutesNum   int
		Imports     codegen.Imports
		Funcs       codegen.Funcs
	}
)

const (
	FILE_NAME_INIT = "gen_init.go"
)

// Generate required integration code
func Generate(pathToServiceDir, tplName, tplOrg, tplVer, basePattern string) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		log.Fatalf("GOPATH is not set")
	}

	rootDir := gopath + string(filepath.Separator) + "src" + string(filepath.Separator) + pathToServiceDir
	tplLoader := template.NewGoPathLoader()
	tplData, err := tplLoader.Load(tplOrg, tplName, tplVer)
	if err != nil {
		log.Fatalf("Could not load template [%s][%s][%s]", tplOrg, tplName, tplVer)
	}

	g := makeDefaultGenerator(string(tplData), rootDir, basePattern)

	g.Generate(rootDir)
}

// Private stuff

func makeDefaultGenerator(tpl, outputPath, basePattern string) *defaultGenerator {

	outputFile := outputPath + string(filepath.Separator) + FILE_NAME_INIT

	return &defaultGenerator{
		tpl:        tpl,
		outputFile: outputFile,
		vm: &viewModel{
			BasePattern: basePattern,
			Imports: codegen.Imports{
				"log": codegen.Import{
					Alias: "log",
					Path:  "github.com/Sirupsen/logrus",
				},
				"net/http": codegen.Import{
					Alias: "",
					Path:  "net/http",
				},
				"errors": codegen.Import{
					Alias: "",
					Path:  "errors",
				},
				"strconv": codegen.Import{
					Alias: "",
					Path:  "strconv",
				},
				"github.com/nildev/lib/router": codegen.Import{
					Alias: "",
					Path:  "github.com/nildev/lib/router",
				},
				"github.com/nildev/lib/utils": codegen.Import{
					Alias: "",
					Path:  "github.com/nildev/lib/utils",
				},
				"github.com/gorilla/mux": codegen.Import{
					Alias: "",
					Path:  "github.com/gorilla/mux",
				},
				"github.com/gorilla/context": codegen.Import{
					Alias: "",
					Path:  "github.com/gorilla/context",
				},
				"github.com/dgrijalva/jwt-go": codegen.Import{
					Alias: "",
					Path:  "github.com/dgrijalva/jwt-go",
				},
			},
			Funcs: codegen.Funcs{},
		},
	}
}

func (dg *defaultGenerator) Generate(pathToServiceDir string) {

	// Open file that we will write all content to
	output, err := os.OpenFile(dg.outputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Could not open output file: %s", err)
	}
	defer func() {
		err := output.Close()
		if err != nil {
			log.Fatal("Could not close file!", err)
		}
	}()

	files, err := ioutil.ReadDir(pathToServiceDir)
	if err != nil {
		log.Fatalf("Could not read dir: %s", pathToServiceDir)
	}

	for _, f := range files {
		err = dg.visit(pathToServiceDir, f)
		if err != nil {
			log.Fatalf("Error while parsing: %s/%s", pathToServiceDir, f.Name())
		}
	}

	if err != nil {
		log.Fatalf("Error while iterating over directory: %s", err)
	}

	if err := codegen.Render(output, dg.tpl, dg.vm); err != nil {
		log.Fatalf("Could not render code: %s", err)
	}
}

func (dg *defaultGenerator) visit(path string, f os.FileInfo) error {
	log.Debugf(" -- Analyse [%s/%s]", path, f.Name())
	if !f.IsDir() {
		if strings.Contains(f.Name(), ".go") && !strings.Contains(f.Name(), FILE_NAME_INIT) {
			dg.analyseFile(path + string(filepath.Separator) + f.Name())
		}
	}

	return nil
}

func (dg *defaultGenerator) analyseFile(pathToFile string) {
	log.Infof("-- [%s] \n", pathToFile)
	fset := token.NewFileSet()
	fast, _ := parser.ParseFile(fset, pathToFile, nil, parser.ParseComments)

	pkgPath := codegen.ParsePackage(fast.Comments)

	if pkgPath == nil {
		return
	}

	dg.vm.PackageName = filepath.Base(*pkgPath)
	dg.vm.RoutesNum = 0

	ast.Inspect(fast, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if ast.IsExported(x.Name.Name) {
				fn := codegen.MakeFunc(x, fast.Imports, fast.Comments)
				if fn != nil {
					dg.vm.RoutesNum++
					dg.vm.Funcs = append(dg.vm.Funcs, *fn)

					for k, v := range fn.In.Imports {
						dg.vm.Imports[k] = v
					}

					for k, v := range fn.Out.Imports {
						dg.vm.Imports[k] = v
					}
				}
			}
		}
		return true
	})
}

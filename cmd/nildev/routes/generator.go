package routes

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nildev/lib/codegen"
	"github.com/nildev/lib/log"
	"github.com/nildev/lib/utils"
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
		RoutesNum   int
		Imports     codegen.Imports
		Services    codegen.Services
	}
)

const (
	FILE_NAME_INIT = "gen_init.go"
)

// Generate required integration code
func Generate(pathToServiceContainerDir string, pathToServices []string, tplName, tplOrg, tplVer string) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		log.Fatalf("GOPATH is not set")
	}

	rootContainerDir := gopath + string(filepath.Separator) + "src" + string(filepath.Separator) + pathToServiceContainerDir
	tplLoader := template.NewGoPathLoader()
	tplData, err := tplLoader.Load(tplOrg, tplName, tplVer)
	if err != nil {
		log.Fatalf("Could not load template [%s][%s][%s]", tplOrg, tplName, tplVer)
	}

	g := makeDefaultGenerator(string(tplData), rootContainerDir)

	g.Generate(pathToServices)
}

func makeDefaultGenerator(tpl, outputPath string) *defaultGenerator {

	outputPath = strings.TrimRight(outputPath, "/")
	outputPath = outputPath + string(filepath.Separator) + "gen"

	ok := utils.Exists(outputPath)
	if !ok {
		if err := os.Mkdir(outputPath, 0777); err != nil {
			log.Fatalf("Could not make dir, %s", err)
		}
	}

	outputFile := outputPath + string(filepath.Separator) + FILE_NAME_INIT

	return &defaultGenerator{
		tpl:        tpl,
		outputFile: outputFile,
		vm: &viewModel{
			PackageName: "gen",
			Imports: codegen.Imports{
				"github.com/nildev/lib/router": codegen.Import{
					Alias: "",
					Path:  "github.com/nildev/lib/router",
				},
			},
			Services: codegen.Services{},
		},
	}
}

func (dg *defaultGenerator) Generate(pathToServices []string) {

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

	lookup := map[string]bool{}
	for _, servicePath := range pathToServices {
		if _, ok := lookup[servicePath]; ok {
			continue
		}
		lookup[servicePath] = true
		dg.vm.Services = append(dg.vm.Services, codegen.Service{
			Import: codegen.Import{
				Alias: "",
				Path:  servicePath,
			},
		})
	}

	dg.vm.RoutesNum = len(lookup)

	if err := codegen.Render(output, dg.tpl, dg.vm); err != nil {
		log.Fatalf("Could not render code: %s", err)
	}
}

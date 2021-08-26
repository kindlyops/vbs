package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
)

type Embed struct {
	Package string
	Embeds  []string
}

func main() {
	output := os.Args[1]
	srcdir := os.Args[2]
	prefix := len(srcdir)
	outdir := path.Dir(output)
	outfile := path.Base(output)
	expanded, err := filepath.Abs(path.Dir(srcdir))
	if err != nil {
		log.Fatalf("Error Abs ing %s: %v", srcdir, err)
	}
	srcdir, err = filepath.EvalSymlinks(expanded)
	if err != nil {
		log.Fatalf("Error EvalSymlinks %s: %v", output, err)
	}
	outfile = path.Join(outdir, outfile)

	manifest := map[string]string{}
	data := Embed{
		Package: path.Base(outdir),
		Embeds:  []string{"dist/*"},
	}

	filepath.WalkDir(srcdir+"/", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		//fmt.Printf("path=%q, isDir=%v\n", path, d.IsDir())
		if len(path) < prefix {
			log.Fatalf("Error walking %s, %s is too short\n", srcdir, path)
		}
		target := path[prefix:]
		if d.IsDir() {
			os.MkdirAll(filepath.Join(outdir, target), 0755)
		} else {

		}
		manifest[target] = fmt.Sprintf("%v", d.IsDir())
		return nil
	})

	// for _, file := range files {
	// 	hash, err := ioutil.ReadFile(file)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// }
	// j, err := json.MarshalIndent(manifest, "", "  ")

	// if err != nil {
	// 	log.Fatalf("Error generating manifest: %s\n", err.Error())
	// }

	t := `
package {{.Package}}

import "embed"

{{range .Embeds}}
//go:embed "{{.}}"
{{end}}
var NextFS embed.FS

func GetNextFS() embed.FS {
	return NextFS
}`
	tmpl, err := template.New("generated").Parse(t)
	if err != nil {
		log.Fatalf("Error parsing template: %s\n", err.Error())
	}
	f, err := os.Create(outfile)
	if err != nil {
		log.Fatalf("Error creating output file: %s\n", err.Error())
	}
	err = tmpl.Execute(f, data)
	if err != nil {
		log.Fatalf("Error executing template: %s\n", err.Error())
	}
	// err = ioutil.WriteFile(output, j, 0644)
	// if err != nil {
	// 	log.Fatalf("Error writing manifest to file: %s\n", err.Error())
	// }
}

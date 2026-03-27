package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Embed struct {
	Package string
	Embeds  []string
}

func main() {
	output := os.Args[1]
	srcdir := os.Args[2]
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
	srcdir += "/"
	distdir := srcdir + "dist/"
	outfile = path.Join(outdir, outfile)

	manifest := map[string]string{}
	data := Embed{
		Package: path.Base(outdir),
		Embeds:  []string{},
	}

	log.Printf("DEBUG scanning %s\n", distdir)

	filepath.WalkDir(distdir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		target := strings.TrimPrefix(path, srcdir)
		if d.IsDir() {
			os.MkdirAll(filepath.Join(outdir, target), 0755)
		} else {
			log.Printf("processing %s\n", target)
			// stinkin https://github.com/golang/go/issues/43854
			data.Embeds = append(data.Embeds, target)
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

import (
	"embed"
	"io/fs"
)

{{range .Embeds}}
//go:embed "{{.}}"
{{end}}

var distDir embed.FS

// DistDirFS contains the embedded dist directory files (without the "dist" prefix)
var DistDirFS, _ = fs.Sub(distDir, "dist")

func GetNextFS() embed.FS {
	return distDir
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

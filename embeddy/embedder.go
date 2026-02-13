package embeddy

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed public
var publicDir embed.FS

// DistDirFS contains the embedded public directory files (without the "public" prefix)
var DistDirFS = MustSubFS(publicDir, "public")

func GetNextFS() embed.FS {
	return publicDir
}

// MustSubFS returns an fs.FS corresponding to the subtree rooted at dir.
// It panics if there's an error.
func MustSubFS(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(fmt.Errorf("failed to create sub FS: %w", err))
	}
	return sub
}

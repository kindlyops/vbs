package embeddy

import (
	"embed"
	"io/fs"
)

//go:embed public
var publicDir embed.FS

// DistDirFS contains the embedded public directory files (without the "public" prefix)
var DistDirFS, _ = fs.Sub(publicDir, "public")

func GetNextFS() embed.FS {
	return publicDir
}

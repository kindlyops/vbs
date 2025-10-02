package embeddy

import (
	"embed"

	"github.com/labstack/echo/v5"
)

//go:embed public
var publicDir embed.FS

// DistDirFS contains the embedded public directory files (without the "public" prefix)
var DistDirFS = echo.MustSubFS(publicDir, "public")

func GetNextFS() embed.FS {
	return publicDir
}

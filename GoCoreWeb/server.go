package main

import (
	"os"
	"path/filepath"

	"github.com/DanielRenne/GoCore/core/app"
)

//GetBinaryPath will return the location of the binary or the project in go run mode.
func main() {
	ex, err := os.Executable()
	if err != nil {
		panic("cant get exe path")
	}
	exPath := filepath.Dir(ex)
	err = app.Initialize(exPath, "webConfig.json")
	app.Run()
}

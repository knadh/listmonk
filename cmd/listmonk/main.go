package main

import (
	"github.com/knadh/listmonk/internal/app"
)

var (
	buildString   string
	versionString string
)

func main() {
	app.Run(buildString, versionString)
}

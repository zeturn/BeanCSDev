package web

import (
	"embed"
)

//go:embed dist/*
var distFS embed.FS

func IndexHTML() ([]byte, error) {
	return distFS.ReadFile("dist/index.html")
}

func Asset(path string) ([]byte, error) {
	return distFS.ReadFile("dist/" + path)
}

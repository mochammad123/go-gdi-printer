//go:build nobundle

package main

import (
	"embed"
	"print-service/engine"
)

//go:embed assets/*
var embeddedAssetsFS embed.FS

// BuildMode indicates the current compile-time template bundle mode
const BuildMode = "Pisah / External Templates (No-Bundle)"

func initEmbedded() {
	engine.RegisterEmbeddedTemplates(nil, "")
	engine.RegisterEmbeddedAssets(embeddedAssetsFS)
}

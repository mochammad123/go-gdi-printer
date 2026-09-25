//go:build !nobundle

package main

import (
	"embed"
	"print-service/engine"
)

//go:embed template/*.json
var embeddedTemplatesFS embed.FS

//go:embed assets/*
var embeddedAssetsFS embed.FS

// BuildMode indicates the current compile-time template bundle mode
const BuildMode = "Full Bundle (Embedded Templates)"

func initEmbedded() {
	engine.RegisterEmbeddedTemplates(embeddedTemplatesFS, "template")
	engine.RegisterEmbeddedAssets(embeddedAssetsFS)
}

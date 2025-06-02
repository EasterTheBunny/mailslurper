package web

import "embed"

//go:embed www/*
var StaticWebAssets embed.FS

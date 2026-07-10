package api

import "embed"

//go:embed ui/dist/*
var DashboardFS embed.FS

package web

import "embed"

// Templates dan static assets dashboard (stylesheet orisinal Gem Hunter).

//go:embed templates/*.html static/css/*.css
var FS embed.FS

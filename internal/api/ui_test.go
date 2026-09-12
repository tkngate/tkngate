package api

import (
	"io/fs"
	"strings"
	"testing"
)

func TestUI(t *testing.T) {
	subFS, err := fs.Sub(DashboardFS, "ui/dist")
	if err != nil {
		t.Fatal(err)
	}

	// Verify index.html exists
	f, err := subFS.Open("index.html")
	if err != nil {
		t.Fatalf("Failed to open index.html: %v", err)
	}
	f.Close()

	// Verify at least one JS and CSS asset exists
	var hasJS, hasCSS bool
	err = fs.WalkDir(subFS, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			if strings.HasSuffix(path, ".js") {
				hasJS = true
			}
			if strings.HasSuffix(path, ".css") {
				hasCSS = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk assets dir: %v", err)
	}

	if !hasJS {
		t.Error("No JS asset bundle found in ui/dist/assets")
	}
	if !hasCSS {
		t.Error("No CSS asset bundle found in ui/dist/assets")
	}
}

package api

import (
	"fmt"
	"io/fs"
	"testing"
)

func TestUI(t *testing.T) {
	subFS, err := fs.Sub(DashboardFS, "ui/dist")
	if err != nil {
		t.Fatal(err)
	}
	f, err := subFS.Open("assets/index-_0NaRAgg.js")
	if err != nil {
		t.Logf("Failed to open file: %v", err)
		
		// Let's print what IS in there
		fs.WalkDir(subFS, ".", func(path string, d fs.DirEntry, err error) error {
			t.Logf("FOUND IN SUBFS: %s", path)
			return nil
		})
		t.Fail()
	} else {
		t.Log("SUCCESSFULLY OPENED FILE!")
		f.Close()
	}
}

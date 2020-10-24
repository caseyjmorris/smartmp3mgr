package internal

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

func getFixturePath(fixture string) string {
	return path.Join(getRootDir(), "testFiles", fixture)
}

func getRootDir() string {
	wd, _ := os.Getwd()
	for !strings.HasSuffix(wd, "smartmp3mgr") {
		wd = filepath.Dir(wd)
	}

	return wd
}

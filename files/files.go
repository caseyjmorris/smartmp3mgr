package files

import (
	"os"
	"path/filepath"
	"strings"
)

func FindMP3Files(root string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// given that we're just enumerating we don't actually need to care about FS errors here... not our problem,
			// FS is corrupt or something.
			return nil
		}

		ext := filepath.Ext(path)
		absolutePath, err := filepath.Abs(path)

		if err == nil && !info.IsDir() && strings.EqualFold(ext, ".mp3") {
			files = append(files, absolutePath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return files, nil
}

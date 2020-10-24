package internal

import (
	"encoding/hex"
	"github.com/bogem/id3v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func ParseMP3(mp3Path string) (Song, error) {
	tags, err := id3v2.Open(mp3Path, id3v2.Options{Parse: true})
	if err != nil {
		return Song{}, err
	}
	song := Song{Path: mp3Path, Artist: tags.Artist(), Album: tags.Album(), Genre: tags.Genre(), Title: tags.Title()}
	err = tags.Close()
	if err != nil {
		return song, err
	}

	mp3Bytes, err := ioutil.ReadFile(mp3Path)

	if err != nil {
		return song, err
	}

	hash, err := Hash(mp3Bytes)
	if err != nil {
		return song, err
	}
	str := hex.EncodeToString(hash[:])

	song.Hash = str

	return song, nil
}

func FindMP3Files(root string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// given that we're just enumerating we don't actually need to care about FS errors here... not our problem,
			// FS is corrupt or something.
			return nil
		}

		ext := filepath.Ext(path)

		if !info.IsDir() && strings.EqualFold(ext, ".mp3") {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return files, nil
}

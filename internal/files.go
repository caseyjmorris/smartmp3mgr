package internal

import (
	"encoding/hex"
	"fmt"
	"github.com/dhowden/tag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func ParseMP3(mp3Path string) (Song, error) {
	file, err := os.OpenFile(mp3Path, os.O_RDONLY, 0)
	defer file.Close()
	if err != nil {
		return Song{}, fmt.Errorf("error opening %q:  %s", mp3Path, err)
	}
	tags, err := tag.ReadFrom(file)
	var song Song
	if err == nil {
		trackNumber, tracks := tags.Track()
		discNumber, discs := tags.Disc()
		song = Song{Path: mp3Path, Artist: tags.Artist(), Album: tags.Album(), Genre: tags.Genre(),
			Title: tags.Title(), TrackNumber: trackNumber, TotalTracks: tracks, DiscNumber: discNumber, TotalDiscs: discs,
			AlbumArtist: tags.AlbumArtist()}
	} else {
		song = Song{}
	}

	mp3Bytes, err := ioutil.ReadFile(mp3Path)

	if err != nil {
		return song, fmt.Errorf("error reading %q:  %s", mp3Path, err)
	}

	hash, err := Hash(mp3Bytes)
	if err != nil {
		return song, fmt.Errorf("error finding hash of %q:  %s", mp3Path, err)
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

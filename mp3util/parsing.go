package mp3util

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/dhowden/tag"
	"os"
)

func ParseMP3(mp3Path string) (Song, error) {
	file, err := os.OpenFile(mp3Path, os.O_RDONLY, 0)
	defer file.Close()
	song := Song{Path: mp3Path}
	if err != nil {
		return song, fmt.Errorf("error opening %q:  %s", mp3Path, err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(file)
	if err != nil {
		return song, fmt.Errorf("error reading %q:  %s", mp3Path, err)
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return song, fmt.Errorf("error seeking file %q:  %s", mp3Path, err)
	}

	tags, err := tag.ReadFrom(file)
	if err == nil {
		trackNumber, tracks := tags.Track()
		discNumber, discs := tags.Disc()
		song = Song{Path: mp3Path, Artist: tags.Artist(), Album: tags.Album(), Genre: tags.Genre(),
			Title: tags.Title(), TrackNumber: trackNumber, TotalTracks: tracks, DiscNumber: discNumber, TotalDiscs: discs,
			AlbumArtist: tags.AlbumArtist()}
	}
	if err != nil {
		return song, fmt.Errorf("error copying buffer for %q:  %s", mp3Path, err)
	}

	hash, err := Hash(buf.Bytes())
	if err != nil {
		return song, fmt.Errorf("error finding hash of %q:  %s", mp3Path, err)
	}
	str := hex.EncodeToString(hash[:])

	song.Hash = str

	return song, nil
}

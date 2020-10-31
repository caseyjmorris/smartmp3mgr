package mp3util

import (
	"encoding/hex"
	"fmt"
	"github.com/dhowden/tag"
	"io/ioutil"
	"os"
)

func ParseMP3(mp3Path string) (Song, error) {
	file, err := os.OpenFile(mp3Path, os.O_RDONLY, 0)
	defer file.Close()
	song := Song{Path: mp3Path}
	if err != nil {
		return song, fmt.Errorf("error opening %q:  %s", mp3Path, err)
	}
	tags, err := tag.ReadFrom(file)
	if err == nil {
		trackNumber, tracks := tags.Track()
		discNumber, discs := tags.Disc()
		song = Song{Path: mp3Path, Artist: tags.Artist(), Album: tags.Album(), Genre: tags.Genre(),
			Title: tags.Title(), TrackNumber: trackNumber, TotalTracks: tracks, DiscNumber: discNumber, TotalDiscs: discs,
			AlbumArtist: tags.AlbumArtist()}
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

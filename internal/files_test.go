package internal

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestFindMP3Files(t *testing.T) {
	path := getFixturePath("")
	result, err := FindMP3Files(path)
	if err != nil {
		t.Error(err)
	}
	var baseNames []string
	for _, el := range result {
		baseNames = append(baseNames, filepath.Base(el))
	}
	sort.Strings(baseNames)
	expected := []string{"spring-chicken.mp3", "wakka-wakka-altered-tags.mp3", "wakka-wakka-default.mp3",
		"wakka-wakka-no-tags.mp3", "wakka-wakka-with-id3v1.mp3"}
	if !reflect.DeepEqual(baseNames, expected) {
		t.Errorf("Elements did not match.  \r\nExpected:  %v  \r\nFound:  %v", expected, result)
	}
}

func TestParseMP3(t *testing.T) {
	path := getFixturePath("wakka-wakka-altered-tags.mp3")
	result, err := ParseMP3(path)
	if err != nil {
		t.Error(err)
	}
	expected := Song{Path: path, Artist: "Thanks Bryan Teoh!", Album: "Thanks FreePD Music!",
		Title: "Wakka Wakka wakkaa", Hash: "883a8beab2a44c5bfcd637855eec3e4b1c89232cb1e1bb17d8cccf9e82c87ecf",
		TrackNumber: 1, DiscNumber: 1}
	if result != expected {
		t.Errorf("Elements did not match.  \r\nExpected:  %v  \r\nFound:  %v", expected, result)
	}
}

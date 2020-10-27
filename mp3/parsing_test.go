package mp3

import (
	"github.com/caseyjmorris/smartmp3mgr/testHelpers"
	"testing"
)

func TestParseMP3(t *testing.T) {
	path := testHelpers.GetFixturePath("wakka-wakka-altered-tags.mp3")
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

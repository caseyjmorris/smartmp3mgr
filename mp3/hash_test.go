package mp3

import (
	"github.com/caseyjmorris/smartmp3mgr/testHelpers"
	"io/ioutil"
	"testing"
)

func TestHashDiffers(t *testing.T) {
	wakka := testHelpers.GetFixturePath("wakka-wakka-default.mp3")
	spring := testHelpers.GetFixturePath("spring-chicken.mp3")
	wakkaBytes, _ := ioutil.ReadFile(wakka)
	springBytes, _ := ioutil.ReadFile(spring)
	wakkaHash, _ := Hash(wakkaBytes)
	springHash, _ := Hash(springBytes)

	if wakkaHash == springHash {
		t.Errorf("Both hashes evaluated to %x", wakkaHash)
	}
}

func TestHashSame(t *testing.T) {
	originalPath := testHelpers.GetFixturePath("wakka-wakka-default.mp3")
	noTagPath := testHelpers.GetFixturePath("wakka-wakka-no-tags.mp3")
	alteredTagPath := testHelpers.GetFixturePath("wakka-wakka-altered-tags.mp3")
	id3v1Tagpath := testHelpers.GetFixturePath("wakka-wakka-with-id3v1.mp3")
	originalBytes, err := ioutil.ReadFile(originalPath)
	if err != nil {
		t.Error(err)
		return
	}
	noTagBytes, err := ioutil.ReadFile(noTagPath)
	if err != nil {
		t.Error(err)
		return
	}

	alteredBytes, err := ioutil.ReadFile(alteredTagPath)
	if err != nil {
		t.Error(err)
		return
	}

	id3v1Bytes, _ := ioutil.ReadFile(id3v1Tagpath)
	originalHash, _ := Hash(originalBytes)
	noTagHash, _ := Hash(noTagBytes)
	alteredHash, _ := Hash(alteredBytes)
	id3v1Hash, _ := Hash(id3v1Bytes)

	if originalHash != noTagHash {
		t.Errorf("Original (%x) did not match no-tag (%x)", originalHash, noTagHash)
	}

	if originalHash != alteredHash {
		t.Errorf("Original (%x) did not match altered (%x)", originalHash, alteredHash)
	}

	if id3v1Hash != originalHash {
		t.Errorf("Original (%x) did not match ID3v1 (%x)", originalHash, id3v1Hash)
	}
}

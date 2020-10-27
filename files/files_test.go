package files

import (
	"github.com/caseyjmorris/smartmp3mgr/testHelpers"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestFindMP3Files(t *testing.T) {
	path := testHelpers.GetFixturePath("")
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

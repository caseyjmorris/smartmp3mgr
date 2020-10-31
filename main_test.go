package main

import (
	"bytes"
	"fmt"
	"github.com/caseyjmorris/smartmp3mgr/mp3fileutil"
	"github.com/caseyjmorris/smartmp3mgr/mp3util"
	"github.com/caseyjmorris/smartmp3mgr/records"
	"github.com/caseyjmorris/smartmp3mgr/testHelpers"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

type testProgressBar struct {
	added int
}

func (t *testProgressBar) Add(num int) error {
	t.added += 1
	return nil
}

func newTestProgressBar(max int64, description ...string) progressReporter {
	return new(testProgressBar)
}

func TestSum(t *testing.T) {
	path := testHelpers.GetFixturePath("")
	mp3s, _ := mp3fileutil.FindMP3Files(path)
	expected := new(bytes.Buffer)
	for _, track := range mp3s {
		fileBytes, _ := ioutil.ReadFile(track)
		hash, _ := mp3util.Hash(fileBytes)
		fmt.Fprintf(expected, "%q:  %x\n", track, hash)
	}
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	sum(stdout, stderr, sumArgs{20, path})
	res := stdout.String()
	expectedS := expected.String()
	if res != expectedS {
		t.Error(res)
	}
}

func TestRecord(t *testing.T) {
	path := testHelpers.GetFixturePath("")
	dbf, err := ioutil.TempFile(os.TempDir(), "smartmp3mgr*.sql")
	if err != nil {
		t.Error("Couldn't make DB temp file")
		return
	}
	dbPath := dbf.Name()
	_ = dbf.Close()
	defer os.Remove(dbPath)
	args := recordArgs{
		directory:           path,
		dbPath:              dbPath,
		reparse:             false,
		degreeOfParallelism: 20,
	}

	for i := 0; i < 4; i++ {
		record(os.Stdout, os.Stderr, newTestProgressBar, args)
	}

	rk, _ := records.Open(dbPath)

	res, _ := rk.FetchSongs()
	var baseNamesOnly []string
	expected := []string{"spring-chicken.mp3", "wakka-wakka-altered-tags.mp3", "wakka-wakka-default.mp3",
		"wakka-wakka-no-tags.mp3", "wakka-wakka-with-id3v1.mp3"}
	for _, r := range res {
		baseName := filepath.Base(r.Path)
		baseNamesOnly = append(baseNamesOnly, baseName)
	}
	sort.Strings(baseNamesOnly)
	if !reflect.DeepEqual(expected, baseNamesOnly) {
		t.Errorf("Records returned don't match.  \r\nExpected:  %+v  \r\nActual:  %+v", expected, baseNamesOnly)
	}
}

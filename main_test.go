package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
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

func TestFindNew(t *testing.T) {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	h := hex.EncodeToString(b[:7])
	tmpPath := filepath.Join(os.TempDir(), "test"+h)
	os.Mkdir(tmpPath, 0755)
	path := testHelpers.GetFixturePath("")
	dbf, err := ioutil.TempFile(os.TempDir(), "smartmp3mgr*.sql")
	if err != nil {
		t.Error("Couldn't make DB temp file")
		return
	}
	dbPath := dbf.Name()
	_ = dbf.Close()
	defer os.Remove(dbPath)
	mp3Path := func(i int) string { return filepath.Join(tmpPath, fmt.Sprintf("%d.mp3", i)) }
	for i := 1; i < 5; i++ {
		defer os.Remove(mp3Path(i))
	}
	defer os.Remove(tmpPath)
	copyFile(testHelpers.GetFixturePath("spring-chicken.mp3"), mp3Path(1), t)
	copyFile(testHelpers.GetFixturePath("wakka-wakka-altered-tags.mp3"), mp3Path(2), t)
	writeRandomFile(mp3Path(3), t)
	writeRandomFile(mp3Path(4), t)

	recordArgs := recordArgs{
		directory:           path,
		dbPath:              dbPath,
		reparse:             false,
		degreeOfParallelism: 20,
	}
	record(os.Stdout, os.Stderr, newTestProgressBar, recordArgs)

	findNewArgs := findNewArgs{
		directory: tmpPath,
		dbPath:    dbPath,
		rehash:    false,
	}

	var res []string

	findNew(os.Stdout, os.Stderr, newTestProgressBar, findNewArgs, &res)
	expected := []string{mp3Path(3), mp3Path(4)}

	if !reflect.DeepEqual(expected, res) {
		t.Errorf("Values differed.  \nExpected:  \n%+v\n\nFound:  \n%+v", expected, res)
	}
}

func writeRandomFile(to string, t *testing.T) {
	b := make([]byte, 1024)
	_, err := rand.Read(b)
	if err != nil {
		t.Error(err)
		return
	}
	err = ioutil.WriteFile(to, b, 0755)
	if err != nil {
		t.Error(err)
	}
}

func copyFile(from string, to string, t *testing.T) {
	b, err := ioutil.ReadFile(from)
	if err != nil {
		t.Error(err)
	}
	err = ioutil.WriteFile(to, b, 0755)
	if err != nil {
		t.Error(err)
	}
}

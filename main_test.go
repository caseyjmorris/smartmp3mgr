package main

import (
	"bytes"
	"fmt"
	"github.com/caseyjmorris/smartmp3mgr/files"
	"github.com/caseyjmorris/smartmp3mgr/mp3"
	"github.com/caseyjmorris/smartmp3mgr/testHelpers"
	"io/ioutil"
	"testing"
)

func TestSum(t *testing.T) {
	path := testHelpers.GetFixturePath("")
	mp3s, _ := files.FindMP3Files(path)
	expected := new(bytes.Buffer)
	for _, track := range mp3s {
		fileBytes, _ := ioutil.ReadFile(track)
		hash, _ := mp3.Hash(fileBytes)
		fmt.Fprintf(expected, "%q:  %x\n", track, hash)
	}
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	sum(stdout, stderr, mp3s)
	res := stdout.String()
	expectedS := expected.String()
	if res != expectedS {
		t.Error(res)
	}
}

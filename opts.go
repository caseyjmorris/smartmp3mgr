package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
)

var findNewCmd = flag.NewFlagSet("find-new", flag.ExitOnError)
var recordCmd = flag.NewFlagSet("record", flag.ExitOnError)
var homeDir, _ = os.UserHomeDir()
var defaultDb = filepath.Join(homeDir, ".smartmp3mgr.sql")

type findNewArgs struct {
	directory           string
	dbPath              string
	rehash              bool
	degreeOfParallelism int
	foldersOnly         bool
}

type recordArgs struct {
	degreeOfParallelism int
	directory           string
	dbPath              string
	reparse             bool
}

func parseFindNewArgs() (result findNewArgs, err error) {
	newCmdDir := findNewCmd.String("directory", "", "directory")
	newCmdDb := findNewCmd.String("dbPath", defaultDb, "path to sqlite db")
	rehash := findNewCmd.Bool("rehash", false, "force a recalculation of existing file hashes")
	dop := findNewCmd.Int("dop", 20, "degree of parallelism")
	foldersOnly := findNewCmd.Bool("fo", false, "show folders only")
	err = findNewCmd.Parse(os.Args[2:])
	if err != nil {
		return
	}

	result = findNewArgs{*newCmdDir, *newCmdDb, *rehash, *dop, *foldersOnly}
	return
}

func parseRecordArgs() (result recordArgs, err error) {
	recordDir := recordCmd.String("directory", "", "directory")
	recordDb := recordCmd.String("dbPath", defaultDb, "path to sqlite db")
	rehash := recordCmd.Bool("reparse", false, "force a rehash of reparse files")
	dop := recordCmd.Int("dop", 20, "degree of parallelism")
	err = recordCmd.Parse(os.Args[2:])
	if err == nil && *dop < 1 {
		err = errors.New("dop must be greater than zero")
	}
	if err != nil {
		return
	}

	result = recordArgs{
		degreeOfParallelism: *dop,
		directory:           *recordDir,
		dbPath:              *recordDb,
		reparse:             *rehash,
	}
	return
}

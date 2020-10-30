package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
)

var sumCmd = flag.NewFlagSet("sum", flag.ExitOnError)
var findNewCmd = flag.NewFlagSet("find-new", flag.ExitOnError)
var recordCmd = flag.NewFlagSet("record", flag.ExitOnError)
var homeDir, _ = os.UserHomeDir()
var defaultDb = filepath.Join(homeDir, ".smartmp3mgr.sql")

type sumArgs struct {
	degreeOfParallelism int
	directory           string
}

type findNewArgs struct {
	directory string
	dbPath    string
	rehash    bool
}

type recordArgs struct {
	degreeOfParallelism int
	directory           string
	dbPath              string
	reparse             bool
}

func parseSumArgs() (result sumArgs, err error) {
	dop := sumCmd.Int("dop", 20, "degree of parallelism")
	d := sumCmd.String("directory", "", "directory for files to sum")
	err = sumCmd.Parse(os.Args[2:])
	if err == nil && *dop < 1 {
		err = errors.New("dop must be greater than zero")
	}
	if err != nil {
		return
	}

	result = sumArgs{
		degreeOfParallelism: *dop,
		directory:           *d,
	}

	return
}

func parseFindNewArgs() (result findNewArgs, err error) {
	newCmdDir := findNewCmd.String("directory", "", "directory")
	newCmdDb := findNewCmd.String("dbPath", defaultDb, "path to sqlite db")
	rehash := findNewCmd.Bool("rehash", false, "force a recalculation of existing file hashes")
	err = findNewCmd.Parse(os.Args[2:])
	if err != nil {
		return
	}

	result = findNewArgs{*newCmdDir, *newCmdDb, *rehash}
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

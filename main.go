package main

import (
	"flag"
	"fmt"
	"github.com/caseyjmorris/smartmp3mgr/files"
	"github.com/caseyjmorris/smartmp3mgr/mp3"
	"github.com/caseyjmorris/smartmp3mgr/records"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:  smartmp3mgr (sum|record|find-new) (args)")
		os.Exit(1)
	}

	recordCmd := flag.NewFlagSet("record", flag.ExitOnError)
	recordDir := recordCmd.String("directory", "", "directory")
	homeDir, _ := os.UserHomeDir()
	defaultDb := filepath.Join(homeDir, ".smartmp3mgr.sql")
	recordDb := recordCmd.String("dbPath", defaultDb, "path to sqlite db")

	findNewCmd := flag.NewFlagSet("find-new", flag.ExitOnError)
	newCmdDir := findNewCmd.String("directory", "", "directory")
	newCmdDb := findNewCmd.String("dbPath", defaultDb, "path to sqlite db")

	switch os.Args[1] {
	case "sum":
		for _, file := range os.Args[2:] {
			mp3, err := mp3.ParseMP3(file)
			if err != nil {
				fmt.Println(err)
				fmt.Println("usage:  smartmp3mgr sum [files]")
				os.Exit(1)
			}
			fmt.Printf("%q:  %s\n", file, mp3.Hash)
		}
		os.Exit(0)
	case "record":
		_ = recordCmd.Parse(os.Args[2:])
		info, err := os.Stat(*recordDir)
		if (err != nil && os.IsNotExist(err)) || !info.IsDir() {
			fmt.Printf("%q is not a directory\n", *recordDir)
			if strings.HasSuffix(os.Args[2], "\\\"") {
				fmt.Println("hint:  are you on Windows and using a quoted directory with the trailing backslash?")
			}
			recordCmd.Usage()
			os.Exit(1)
		}

		files, err := files.FindMP3Files(*recordDir)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		var parsed []mp3.Song
		for _, file := range files {
			record, err := mp3.ParseMP3(file)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			parsed = append(parsed, record)
		}
		db, err := records.Open(*recordDb)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer db.Close()

		err = db.RecordSongs(parsed)
		if err != nil {
			fmt.Println(err)
			recordCmd.Usage()
			os.Exit(1)
		}

		os.Exit(0)
	case "find-new":
		_ = findNewCmd.Parse(os.Args[2:])
		info, err := os.Stat(*newCmdDir)
		if (err != nil && os.IsNotExist(err)) || !info.IsDir() {
			fmt.Printf("%q is not a directory\n", *newCmdDir)
			if strings.HasSuffix(os.Args[2], "\\\"") {
				fmt.Println("hint:  are you on Windows and using a quoted directory with the trailing backslash?")
			}
			findNewCmd.Usage()
			os.Exit(1)
		}

		files, err := files.FindMP3Files(*newCmdDir)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		var parsed []mp3.Song
		var hashes []string
		for _, file := range files {
			record, err := mp3.ParseMP3(file)
			if err != nil {
				continue
			}
			parsed = append(parsed, record)
			hashes = append(hashes, record.Hash)
		}
		db, err := records.Open(*newCmdDb)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer db.Close()

		existing, err := db.FetchSongs(hashes)
		existsMap := make(map[string]mp3.Song)

		for _, existingRecord := range existing {
			existsMap[existingRecord.Hash] = existingRecord
		}

		uniq := 0

		for _, parsedRecord := range parsed {
			if _, ok := existsMap[parsedRecord.Hash]; ok {
				continue
			}
			fmt.Println(parsedRecord.Path)
			uniq++
		}

		fmt.Printf("(%d new songs)", uniq)

		if err != nil {
			fmt.Println(err)
			findNewCmd.Usage()
			os.Exit(1)
		}

		os.Exit(0)
	default:
		fmt.Println("Usage:  smartmp3mgr (sum|record|find-new) (args)")
		os.Exit(1)
	}
}

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/caseyjmorris/smartmp3mgr/internal"
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
	recordDb := recordCmd.String("dbPath", filepath.Join(homeDir, ".smartmp3mgr.sql"), "path to sqlite db")

	switch os.Args[1] {
	case "sum":
		for _, file := range os.Args[2:] {
			mp3, err := internal.ParseMP3(file)
			if err != nil {
				fmt.Println(err)
				fmt.Println("usage:  smartmp3mgr sum [files]")
				os.Exit(1)
			}
			fmt.Printf("%q:  %s\n", file, mp3.Hash)
		}
		os.Exit(0)
	case "record":
		recordCmd.Parse(os.Args[2:])
		info, err := os.Stat(*recordDir)
		if (err != nil && os.IsNotExist(err)) || !info.IsDir() {
			fmt.Printf("%q is not a directory\n", *recordDir)
			if strings.HasSuffix(os.Args[2], "\\\"") {
				fmt.Println("hint:  are you on Windows and using a quoted directory with the trailing backslash?")
			}
			recordCmd.Usage()

			files, err := internal.FindMP3Files(os.Args[2])

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			var parsed []internal.Song
			for _, file := range files {
				record, err := internal.ParseMP3(file)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				parsed = append(parsed, record)
			}
			db, err := sql.Open("sqlite3", *recordDb)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			defer db.Close()

			err = internal.RecordSongs(db, parsed)
			if err != nil {
				fmt.Println(err)
				recordCmd.Usage()
				os.Exit(1)
			}

			os.Exit(0)
		}
	}
}

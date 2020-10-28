package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/caseyjmorris/smartmp3mgr/files"
	"github.com/caseyjmorris/smartmp3mgr/mp3"
	"github.com/caseyjmorris/smartmp3mgr/records"
	_ "github.com/mattn/go-sqlite3"
	"github.com/schollz/progressbar/v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:  smartmp3mgr (sum|record|find-new) (args)")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "sum":
		sum()
	case "record":
		record()
	case "find-new":
		findNew()
	default:
		fmt.Println("Usage:  smartmp3mgr (sum|record|find-new) (args)")
		os.Exit(1)
	}
}

func sum() {
	for _, file := range os.Args[2:] {
		track, err := mp3.ParseMP3(file)
		if err != nil {
			fmt.Println(err)
			fmt.Println("usage:  smartmp3mgr sum [files]")
			os.Exit(1)
		}
		fmt.Printf("%q:  %s\n", file, track.Hash)
	}
	os.Exit(0)
}

func record() {
	recordCmd := flag.NewFlagSet("record", flag.ExitOnError)
	recordDir := recordCmd.String("directory", "", "directory")
	homeDir, _ := os.UserHomeDir()
	defaultDb := filepath.Join(homeDir, ".smartmp3mgr.sql")
	recordDb := recordCmd.String("dbPath", defaultDb, "path to sqlite db")
	rehash := recordCmd.Bool("reparse", false, "force a rehash of reparse files")

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

	fmt.Printf("Scanning %q for MP3s\n", *recordDir)
	mp3Files, err := files.FindMP3Files(*recordDir)

	db, err := records.Open(*recordDb)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	existing, err := db.FetchSongs()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	existingMap := make(map[string]mp3.Song)

	if !*rehash {
		for _, existingFile := range existing {
			existingMap[existingFile.Path] = existingFile
		}
	}

	fmt.Printf("Scanning %d files\n", len(mp3Files))

	bar := progressbar.Default(int64(len(mp3Files)))

	var parsed []mp3.Song
	for _, file := range mp3Files {
		var record mp3.Song
		if cached, ok := existingMap[file]; ok {
			record = cached
		} else {
			record, err = mp3.ParseMP3(file)
			if err != nil {
				continue
			}
		}
		parsed = append(parsed, record)
		_ = bar.Add(1)
	}

	tx, err := db.Begin()

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error opening transaction:  %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updating database at %q\n", *recordDb)

	bar = progressbar.Default(int64(len(parsed)))
	for _, parsedSong := range parsed {
		if _, ok := existingMap[parsedSong.Path]; ok {
			_ = bar.Add(1)
			continue
		}
		err = db.RecordSong(parsedSong)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error saving %q:  %s\n", parsedSong.Path, err)
			os.Exit(1)
		}
		_ = bar.Add(1)
	}

	err = tx.Commit()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error commiting transaction:  %s\n", err)
		os.Exit(1)
	}
	err = db.Close()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error closign db:  %s\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func findNew() {
	homeDir, _ := os.UserHomeDir()
	defaultDb := filepath.Join(homeDir, ".smartmp3mgr.sql")
	findNewCmd := flag.NewFlagSet("find-new", flag.ExitOnError)
	newCmdDir := findNewCmd.String("directory", "", "directory")
	newCmdDb := findNewCmd.String("dbPath", defaultDb, "path to sqlite db")
	rehash := findNewCmd.Bool("rehash", false, "force a recalculation of existing file hashes")
	err := findNewCmd.Parse(os.Args[2:])
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	db, err := records.Open(*newCmdDb)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	defer db.Close()

	knownHashes, err := db.GetHashes()

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to open db %q:  %s", *newCmdDb, err)
	}

	info, err := os.Stat(*newCmdDir)
	if (err != nil && os.IsNotExist(err)) || !info.IsDir() {
		fmt.Printf("%q is not a directory\n", *newCmdDir)
		if strings.HasSuffix(os.Args[2], "\\\"") {
			fmt.Println("hint:  are you on Windows and using a quoted directory with the trailing backslash?")
		}
		findNewCmd.Usage()
		os.Exit(1)
	}

	fmt.Printf("Looking for files in %q\n", *newCmdDir)

	mp3Files, err := files.FindMP3Files(*newCmdDir)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	existsMap := make(map[string]mp3.Song)
	if !*rehash {
		fmt.Printf("Checking existing records in DB %q\n", *newCmdDb)
		existingFiles, err := db.FetchSongs()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error reading database:  %s\n", err)
		}
		for _, existingRecord := range existingFiles {
			existsMap[existingRecord.Hash] = existingRecord
		}
	}
	uniq := 0

	tx, err := db.Begin()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to start transaction:  %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Hashing %d files and comparing against existing records in DB %q\n", len(mp3Files), *newCmdDb)

	bar := progressbar.Default(int64(len(mp3Files)))
	var results []string

	for _, file := range mp3Files {
		var hashS string
		if existing, ok := knownHashes[file]; ok {
			hashS = existing
		} else {
			bytes, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			hash, err := mp3.Hash(bytes)
			if err != nil {
				continue
			}
			hashS = hex.EncodeToString(hash[:])
			err = db.CacheHash(file, hashS)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "failed to write cached hash:  %s\n", err)
				os.Exit(1)
			}
		}

		if _, ok := existsMap[hashS]; !ok {
			uniq++
			results = append(results, file)
		}
		_ = bar.Add(1)
	}

	err = tx.Commit()

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to commit transaction:  %s\n", err)
		os.Exit(1)
	}

	for _, result := range results {
		fmt.Println(result)
	}

	fmt.Printf("(%d new songs)\n", uniq)

	os.Exit(0)
}

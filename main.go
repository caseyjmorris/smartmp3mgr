package main

import (
	"encoding/hex"
	"fmt"
	"github.com/caseyjmorris/smartmp3mgr/files"
	"github.com/caseyjmorris/smartmp3mgr/mp3"
	"github.com/caseyjmorris/smartmp3mgr/records"
	_ "github.com/mattn/go-sqlite3"
	"github.com/schollz/progressbar/v3"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:  smartmp3mgr (sum|record|find-new) (args)")
		os.Exit(1)
	}

	prf := func(max int64, description ...string) progressReporter {
		return progressbar.Default(max, description...)
	}

	switch os.Args[1] {
	case "sum":
		sum(os.Stdout, os.Stdin)
	case "record":
		record(os.Stdout, os.Stderr, prf)
	case "find-new":
		findNew()
	default:
		diePrintln(os.Stderr, "Usage:  smartmp3mgr (sum|record|find-new) (args)")
	}
}

func diePrintf(w io.Writer, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(w, format, args)
	os.Exit(1)
}

func diePrintln(w io.Writer, a ...interface{}) {
	_, _ = fmt.Fprintln(w, a)
	os.Exit(1)
}

func sum(stdout io.Writer, stderr io.Writer) {
	for _, file := range os.Args[2:] {
		track, err := mp3.ParseMP3(file)
		if err != nil {
			diePrintf(stderr, "%s\nusage:  smartmp3mgr sum [files]")
		}
		_, _ = fmt.Fprintf(stdout, "%q:  %s\n", file, track.Hash)
	}
	os.Exit(0)
}

func record(stdout io.Writer, stderr io.Writer, pb progressReporterFactory) {
	args, err := parseRecordArgs()
	if err != nil {
		diePrintf(stderr, "error parsing:  %s", err)
	}

	info, err := os.Stat(args.directory)
	if (err != nil && os.IsNotExist(err)) || !info.IsDir() {
		_, _ = fmt.Fprintf(stderr, "%q is not a directory\n", args.directory)
		if strings.HasSuffix(os.Args[2], "\\\"") {
			_, _ = fmt.Fprintf(stderr, "hint:  are you on Windows and using a quoted directory with the trailing backslash?")
		}
		recordCmd.Usage()
		os.Exit(1)
	}

	_, _ = fmt.Fprintf(stdout, "Scanning %q for MP3s\n", args.directory)
	mp3Files, err := files.FindMP3Files(args.directory)

	db, err := records.Open(args.dbPath)
	if err != nil {
		diePrintln(stderr, err)
	}

	existing, err := db.FetchSongs()
	if err != nil {
		diePrintln(stderr, err)
	}

	existingMap := make(map[string]mp3.Song)

	if !args.reparse {
		for _, existingFile := range existing {
			existingMap[existingFile.Path] = existingFile
		}
	}

	_, _ = fmt.Fprintf(stdout, "Scanning %d files\n", len(mp3Files))

	bar := pb(int64(len(mp3Files)))

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
		diePrintf(stderr, "error opening transaction:  %s\n", err)
	}

	_, _ = fmt.Fprintf(stdout, "Updating database at %q\n", args.dbPath)

	bar = pb(int64(len(parsed)))
	for _, parsedSong := range parsed {
		if _, ok := existingMap[parsedSong.Path]; ok {
			_ = bar.Add(1)
			continue
		}
		err = db.RecordSong(parsedSong)
		if err != nil {
			diePrintf(stderr, "error saving %q:  %s\n", parsedSong.Path, err)
		}
		_ = bar.Add(1)
	}

	err = tx.Commit()
	if err != nil {
		diePrintf(stderr, "error commiting transaction:  %s\n", err)
	}
	err = db.Close()
	if err != nil {
		diePrintf(stderr, "error closign db:  %s\n", err)
	}

	os.Exit(0)
}

func findNew() {
	args, err := parseFindNewArgs()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	db, err := records.Open(args.dbPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	defer db.Close()

	knownHashes, err := db.GetHashes()

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to open db %q:  %s", args.dbPath, err)
	}

	info, err := os.Stat(args.directory)
	if (err != nil && os.IsNotExist(err)) || !info.IsDir() {
		fmt.Printf("%q is not a directory\n", args.directory)
		if strings.HasSuffix(os.Args[2], "\\\"") {
			fmt.Println("hint:  are you on Windows and using a quoted directory with the trailing backslash?")
		}
		findNewCmd.Usage()
		os.Exit(1)
	}

	fmt.Printf("Looking for files in %q\n", args.directory)

	mp3Files, err := files.FindMP3Files(args.directory)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	existsMap := make(map[string]mp3.Song)
	if !args.rehash {
		fmt.Printf("Checking existing records in DB %q\n", args.dbPath)
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

	fmt.Printf("Hashing %d files and comparing against existing records in DB %q\n", len(mp3Files), args.dbPath)

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

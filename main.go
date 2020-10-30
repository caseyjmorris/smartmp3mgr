package main

import (
	"encoding/hex"
	"fmt"
	"github.com/caseyjmorris/smartmp3mgr/mp3fileutil"
	"github.com/caseyjmorris/smartmp3mgr/mp3util"
	"github.com/caseyjmorris/smartmp3mgr/records"
	_ "github.com/mattn/go-sqlite3"
	"github.com/schollz/progressbar/v3"
	"io"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
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
		args, err := parseSumArgs()
		if err != nil {
			diePrintln(os.Stderr, err)
		}
		sum(os.Stdout, os.Stdin, args)
	case "record":
		args, err := parseRecordArgs()
		if err != nil {
			diePrintf(os.Stderr, "error parsing:  %s", err)
		}
		record(os.Stdout, os.Stderr, prf, args)
	case "find-new":
		args, err := parseFindNewArgs()
		if err != nil {
			diePrintf(os.Stderr, "%s", err)
		}
		findNew(os.Stdout, os.Stderr, prf, args)
	default:
		diePrintln(os.Stderr, "Usage:  smartmp3mgr (sum|record|find-new) (args)")
	}

	os.Exit(0)
}

func sum(stdout io.Writer, stderr io.Writer, args sumArgs) {
	if args.degreeOfParallelism < 1 {
		diePrintln(stderr, "Degree of parallelism must be greater than 0")
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

	mp3files, err := mp3fileutil.FindMP3Files(args.directory)
	if err != nil {
		diePrintln(stderr, err)
	}

	q := make(chan string, len(mp3files))
	sink := make(chan struct {
		song mp3util.Song
		err  error
	}, len(mp3files))
	var r []mp3util.Song

	for _, file := range mp3files {
		q <- file
	}
	close(q)

	dop := int(math.Min(float64(args.degreeOfParallelism), float64(len(mp3files))))

	var wg sync.WaitGroup
	wg.Add(len(mp3files))

	for i := 0; i < dop; i++ {
		go func(q <-chan string, sink chan<- struct {
			song mp3util.Song
			err  error
		}) {
			for el := range q {
				res, err := mp3util.ParseMP3(el)
				x := struct {
					song mp3util.Song
					err  error
				}{res, err}
				sink <- x
			}
		}(q, sink)
	}

	go func() {
		for file := range sink {
			if file.err != nil {
				diePrintf(stderr, "%s\nusage:  smartmp3mgr sum [files]", err)
			}
			r = append(r, file.song)
			wg.Done()
		}
	}()

	wg.Wait()

	sort.Slice(r, func(i, j int) bool {
		return r[i].Path < r[j].Path
	})

	for _, track := range r {
		_, _ = fmt.Fprintf(stdout, "%q:  %s\n", track.Path, track.Hash)
	}
}

func record(stdout io.Writer, stderr io.Writer, pb progressReporterFactory, args recordArgs) {
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
	mp3Files, err := mp3fileutil.FindMP3Files(args.directory)

	db, err := records.Open(args.dbPath)
	if err != nil {
		diePrintln(stderr, err)
	}

	existing, err := db.FetchSongs()
	if err != nil {
		diePrintln(stderr, err)
	}

	existingMap := make(map[string]mp3util.Song)

	if !args.reparse {
		for _, existingFile := range existing {
			existingMap[existingFile.Path] = existingFile
		}
	}

	_, _ = fmt.Fprintf(stdout, "Scanning %d files\n", len(mp3Files))

	bar := pb(int64(len(mp3Files)))

	var parsed []mp3util.Song
	for _, file := range mp3Files {
		var record mp3util.Song
		if cached, ok := existingMap[file]; ok {
			record = cached
		} else {
			record, err = mp3util.ParseMP3(file)
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
		diePrintf(stderr, "error committing transaction:  %s\n", err)
	}
	err = db.Close()
	if err != nil {
		diePrintf(stderr, "error closing db:  %s\n", err)
	}
}

func findNew(stdout io.Writer, stderr io.Writer, prf progressReporterFactory, args findNewArgs) {
	db, err := records.Open(args.dbPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s", err)
		os.Exit(1)
	}
	defer db.Close()

	knownHashes, err := db.GetHashes()

	if err != nil {
		_, _ = fmt.Fprintf(stderr, "failed to open db %q:  %s", args.dbPath, err)
	}

	info, err := os.Stat(args.directory)
	if (err != nil && os.IsNotExist(err)) || !info.IsDir() {
		_, _ = fmt.Fprintf(stderr, "%q is not a directory\n", args.directory)
		if strings.HasSuffix(os.Args[2], "\\\"") {
			_, _ = fmt.Fprintln(stderr, "hint:  are you on Windows and using a quoted directory with the trailing backslash?")
		}
		findNewCmd.Usage()
		os.Exit(1)
	}

	_, _ = fmt.Fprintf(stdout, "Looking for files in %q\n", args.directory)

	mp3Files, err := mp3fileutil.FindMP3Files(args.directory)

	if err != nil {
		diePrintln(stdout, err)
	}

	existsMap := make(map[string]mp3util.Song)
	if !args.rehash {
		_, _ = fmt.Fprintf(stdout, "Checking existing records in DB %q\n", args.dbPath)
		existingFiles, err := db.FetchSongs()
		if err != nil {
			diePrintf(stderr, "Error reading database:  %s\n", err)
		}
		for _, existingRecord := range existingFiles {
			existsMap[existingRecord.Hash] = existingRecord
		}
	}
	uniq := 0

	tx, err := db.Begin()
	if err != nil {
		diePrintf(stderr, "failed to start transaction:  %s\n", err)
	}

	_, _ = fmt.Fprintf(stdout, "Hashing %d files and comparing against existing records in DB %q\n", len(mp3Files), args.dbPath)

	bar := prf(int64(len(mp3Files)))
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
			hash, err := mp3util.Hash(bytes)
			if err != nil {
				continue
			}
			hashS = hex.EncodeToString(hash[:])
			err = db.CacheHash(file, hashS)
			if err != nil {
				diePrintf(stderr, "failed to write cached hash:  %s\n", err)
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
		diePrintf(stderr, "failed to commit transaction:  %s\n", err)
	}

	for _, result := range results {
		fmt.Fprintln(stdout, result)
	}

	_, _ = fmt.Fprintf(stdout, "(%d new songs)\n", uniq)
}

func diePrintf(w io.Writer, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(w, format, args...)
	os.Exit(1)
}

func diePrintln(w io.Writer, a ...interface{}) {
	_, _ = fmt.Fprintln(w, a...)
	os.Exit(1)
}

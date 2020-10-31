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

type maybeSong struct {
	song mp3util.Song
	err  error
}

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
		findNew(os.Stdout, os.Stderr, prf, args, nil)
	default:
		diePrintln(os.Stderr, "Usage:  smartmp3mgr (sum|record|find-new) (args)")
	}

	os.Exit(0)
}

func sum(stdout io.Writer, stderr io.Writer, args sumArgs) {
	mp3files, err := mp3fileutil.FindMP3Files(args.directory)
	if err != nil {
		diePrintln(stderr, err)
	}

	q := make(chan string, len(mp3files))
	sink := make(chan maybeSong, len(mp3files))
	var r []mp3util.Song

	for _, file := range mp3files {
		q <- file
	}
	close(q)

	dop := int(math.Min(float64(args.degreeOfParallelism), float64(len(mp3files))))

	var wg sync.WaitGroup
	wg.Add(len(mp3files))

	for i := 0; i < dop; i++ {
		go func(q <-chan string, sink chan<- maybeSong) {
			for el := range q {
				res, err := mp3util.ParseMP3(el)
				sink <- maybeSong{
					song: res,
					err:  err,
				}
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
	dieUnlessDirectoryExists(stderr, args.directory)
	db, existingMap := fetchSongsOrDie(stderr, args.dbPath, args.reparse)

	_, _ = fmt.Fprintf(stdout, "Scanning %q for MP3s\n", args.directory)
	mp3Files, err := mp3fileutil.FindMP3Files(args.directory)

	fileQ := make(chan string, len(mp3Files))

	for _, file := range mp3Files {
		fileQ <- file
	}
	close(fileQ)

	_, _ = fmt.Fprintf(stdout, "Scanning %d files\n", len(mp3Files))

	bar := pb(int64(len(mp3Files)))

	songQ := make(chan mp3util.Song, len(mp3Files))
	doneQ := make(chan int, len(mp3Files))
	var wg sync.WaitGroup
	var wg2 sync.WaitGroup
	for i := 0; i < args.degreeOfParallelism; i++ {
		go func(songQ chan<- mp3util.Song, doneQ chan<- int) {
			for file := range fileQ {
				wg.Add(1)
				wg2.Add(1)
				var record mp3util.Song
				if cached, ok := existingMap[file]; ok {
					record = cached
				} else {
					record, err = mp3util.ParseMP3(file)
					if err != nil {
						continue
					}
				}
				songQ <- record
				doneQ <- 1
			}
		}(songQ, doneQ)
	}

	var ct int64

	go func(doneQ <-chan int) {
		for i := range doneQ {
			ct++
			_ = bar.Add(i)
			wg.Done()
		}
	}(doneQ)

	wg.Wait()

	tx, err := db.Begin()

	if err != nil {
		diePrintf(stderr, "error opening transaction:  %s\n", err)
	}

	_, _ = fmt.Fprintf(stdout, "Updating database at %q\n", args.dbPath)

	bar = pb(ct)

	go func(songQ <-chan mp3util.Song) {
		for s := range songQ {
			if _, ok := existingMap[s.Path]; ok {
				_ = bar.Add(1)
				wg2.Done()
				continue
			}
			err = db.RecordSong(s)
			if err != nil {
				diePrintf(stderr, "error saving %q:  %s\n", s.Path, err)
			}
			_ = bar.Add(1)
			wg2.Done()
		}
	}(songQ)

	wg2.Wait()

	err = tx.Commit()
	if err != nil {
		diePrintf(stderr, "error committing transaction:  %s\n", err)
	}
	err = db.Close()
	if err != nil {
		diePrintf(stderr, "error closing db:  %s\n", err)
	}
}

func findNew(stdout io.Writer, stderr io.Writer, prf progressReporterFactory, args findNewArgs, resultCapture *[]string) {
	dieUnlessDirectoryExists(stderr, args.directory)
	if args.degreeOfParallelism < 1 {
		diePrintln(stderr, "degree of parallelism must be greater than 0")
	}

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

	_, _ = fmt.Fprintf(stdout, "Looking for files in %q\n", args.directory)

	mp3Files, err := mp3fileutil.FindMP3Files(args.directory)

	if err != nil {
		diePrintln(stdout, err)
	}

	fileQ := make(chan string, len(mp3Files))

	for _, f := range mp3Files {
		fileQ <- f
	}
	close(fileQ)

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

	doneQ := make(chan int, len(mp3Files))
	uniqQ := make(chan string)
	fileHashQ := make(chan [2]string)
	var wg sync.WaitGroup
	wg.Add(len(mp3Files))

	for i := 0; i < args.degreeOfParallelism; i++ {
		go func(doneQ chan<- int, uniqQ chan<- string, fileHashQ chan<- [2]string) {
			for file := range fileQ {
				var hashS string
				if existing, ok := knownHashes[file]; ok {
					hashS = existing
				} else {
					bytes, err := ioutil.ReadFile(file)
					if err != nil {
						doneQ <- 1
						wg.Done()
						continue
					}
					hash, err := mp3util.Hash(bytes)
					if err != nil {
						doneQ <- 1
						wg.Done()
						continue
					}
					hashS = hex.EncodeToString(hash[:])
					wg.Add(1)
					fileHashQ <- [2]string{file, hashS}

				}

				if _, ok := existsMap[hashS]; !ok {
					wg.Add(1)
					uniqQ <- file
				}

				doneQ <- 1
				wg.Done()
			}
		}(doneQ, uniqQ, fileHashQ)
	}

	go func() {
		for i := range doneQ {
			_ = bar.Add(i)
		}
	}()

	go func() {
		for u := range uniqQ {
			uniq++
			results = append(results, u)
			if resultCapture != nil {
				*resultCapture = append(*resultCapture, u)
			}
			wg.Done()
		}
	}()

	go func() {
		for fh := range fileHashQ {
			file := fh[0]
			hashS := fh[1]
			err = db.CacheHash(file, hashS)
			if err != nil {
				diePrintf(stderr, "failed to write cached hash:  %s\n", err)
			}
			wg.Done()
		}
	}()

	wg.Wait()

	sort.Strings(results)

	err = tx.Commit()

	if err != nil {
		diePrintf(stderr, "failed to commit transaction:  %s\n", err)
	}

	for _, result := range results {
		fmt.Fprintln(stdout, result)
	}

	_, _ = fmt.Fprintf(stdout, "(%d new songs)\n", uniq)
}

func dieUnlessDirectoryExists(stderr io.Writer, directory string) {
	info, err := os.Stat(directory)
	if (err != nil && os.IsNotExist(err)) || !info.IsDir() {
		_, _ = fmt.Fprintf(stderr, "%q is not a directory\n", directory)
		if strings.HasSuffix(os.Args[2], "\\\"") {
			_, _ = fmt.Fprintf(stderr, "hint:  are you on Windows and using a quoted directory with the trailing backslash?")
		}
		recordCmd.Usage()
		os.Exit(1)
	}
}

func fetchSongsOrDie(stderr io.Writer, dbPath string, reparse bool) (*records.RecordKeeper, map[string]mp3util.Song) {
	db, err := records.Open(dbPath)
	if err != nil {
		diePrintln(stderr, err)
	}

	existing, err := db.FetchSongs()
	if err != nil {
		diePrintln(stderr, err)
	}

	existingMap := make(map[string]mp3util.Song)

	if !reparse {
		for _, existingFile := range existing {
			existingMap[existingFile.Path] = existingFile
		}
	}

	return db, existingMap
}

func diePrintf(w io.Writer, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(w, format, args...)
	os.Exit(1)
}

func diePrintln(w io.Writer, a ...interface{}) {
	_, _ = fmt.Fprintln(w, a...)
	os.Exit(1)
}

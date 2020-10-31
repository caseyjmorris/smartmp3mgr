# Smart MP3 Manager

## About

This is largely an excuse to learn the Go programming language (the curious could probably map the commit history to
the order concepts are introduced in the [book of the same name](https://www.gopl.io/)), but it does solve a problem I
had:  otherwise identical MP3 files with different tags being in multiple places.  While it should work on OS X or 
Linux, it has only been tested on Windows.

There are two basic functions in this program:  `record`, which scans a "canonical" folder (or folders) for MP3s and
records their tags and hashes in an SQLite database, and `find-new`, which will look at a non-canonical folder, find
hashes, and compare the hashes to the SQLite database to determine while files in this folder are not duplicates.

## Setup

`go build -o c:\some\folder\on\PATH` should be sufficient.  Because of the dependency on 
[go-sqlite3](https://github.com/mattn/go-sqlite3), you may need to install gcc, if you don't  have it.

## Basic usage

```
smartmp3mgr record -directory c:\mymusic
smartmp3mgr find-new -directory c:\unsortedmusic
```

More detailed information is available with the `-help` parameter to these commands (e.g., `smartmp3mgr record -help`).

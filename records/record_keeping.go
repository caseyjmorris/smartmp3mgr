package records

import (
	"database/sql"
	"fmt"
	"github.com/caseyjmorris/smartmp3mgr/internal"
	_ "github.com/mattn/go-sqlite3"
)

type RecordKeeper struct {
	*sql.DB
}

func Open(connectionString string) (*RecordKeeper, error) {
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite db:  %s", err)
	}
	return &RecordKeeper{db}, nil
}

func (rk *RecordKeeper) prepareSongsTable() error {
	const statement = `
		CREATE TABLE IF NOT EXISTS 
		  Songs (Path TEXT NOT NULL PRIMARY KEY, Artist TEXT, Album TEXT, Title TEXT, Hash TEXT, Genre TEXT,
		  AlbumArtist TEXT, TrackNumber INTEGER, TotalTracks INTEGER, DiscNumber INTEGER, TotalDiscs INTEGER);
		CREATE INDEX IF NOT EXISTS
		  SongsHashIndex ON Songs(Hash)
    `

	_, err := rk.Exec(statement)

	if err != nil {
		return err
	}

	return nil
}

func (rk *RecordKeeper) RecordSongs(songs []internal.Song) error {
	err := rk.prepareSongsTable()
	if err != nil {
		return err
	}

	tx, err := rk.Begin()
	if err != nil {
		return err
	}

	const insertStatement = `
		INSERT INTO Songs(Path, Artist, Album, Title, Hash, Genre, AlbumArtist, TrackNumber, TotalTracks, 
		  DiscNumber, TotalDiscs)
		VALUES (@Path, @Artist, @Album, @Title, @Hash, @Genre, @AlbumArtist, @TrackNumber, @TotalTracks, 
		@DiscNumber, @TotalDiscs)
		ON CONFLICT(Path) DO UPDATE SET Path = @Path, Artist = @Artist, Album = @Album, Title = @Title, Hash = @Hash,
		Genre = @Genre, AlbumArtist = @AlbumArtist, TrackNumber = @TrackNumber, TotalTracks = @TotalTracks,
		DiscNumber = @DiscNumber, TotalDiscs = @TotalDiscs
		`

	insertPrepared, err := rk.Prepare(insertStatement)
	if err != nil {
		return err
	}

	for _, song := range songs {
		_, err = insertPrepared.Exec(song.Path, song.Artist, song.Album, song.Title, song.Hash, song.Genre,
			song.AlbumArtist, song.TrackNumber, song.TotalTracks, song.DiscNumber, song.TotalDiscs)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (rk *RecordKeeper) FetchSongs(desiredHashes []string) ([]internal.Song, error) {
	var result []internal.Song

	err := rk.prepareSongsTable()
	if err != nil {
		return result, err
	}

	const query = `
		SELECT Path, Artist, Album, Title, Hash, Genre, AlbumArtist, TrackNumber, TotalTracks, DiscNumber, TotalDiscs 
        FROM Songs
		`

	const filteredQuery = `
		SELECT Path, Artist, Album, Title, s.Hash, Genre, AlbumArtist, TrackNumber, TotalTracks, DiscNumber, TotalDiscs 
        FROM Songs s
        INNER JOIN DesiredHashes dh ON dh.Hash = s.Hash 
		`

	var rows *sql.Rows

	if len(desiredHashes) == 0 {
		rows, err = rk.Query(query)
	} else {
		tx, err := rk.Begin()
		if err != nil {
			return result, err
		}

		_, err = rk.Exec("CREATE TEMPORARY TABLE DesiredHashes(hash TEXT NOT NULL)")
		if err != nil {
			return result, err
		}

		const insertSQL = `
			INSERT INTO DesiredHashes(hash) VALUES (?)
			`

		stmt, err := rk.Prepare(insertSQL)
		if err != nil {
			return result, err
		}

		for _, hash := range desiredHashes {
			_, err = stmt.Exec(hash)
			if err != nil {
				return result, err
			}
		}

		err = tx.Commit()
		if err != nil {
			return result, err
		}

		rows, err = rk.Query(filteredQuery)
		if err != nil {
			return result, err
		}
	}

	defer rows.Close()
	if err != nil {
		return result, nil
	}

	for rows.Next() {
		var song internal.Song
		err = rows.Scan(&song.Path, &song.Artist, &song.Album, &song.Title, &song.Hash, &song.Genre, &song.AlbumArtist,
			&song.TrackNumber, &song.TotalTracks, &song.DiscNumber, &song.TotalDiscs)
		if err != nil {
			return result, err
		}
		result = append(result, song)
	}

	return result, nil
}

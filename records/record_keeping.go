package records

import (
	"database/sql"
	"fmt"
	"github.com/caseyjmorris/smartmp3mgr/internal"
	_ "github.com/mattn/go-sqlite3"
)

type RecordKeeper struct {
	*sql.DB
	preparedStatementCache map[string]*sql.Stmt
}

func Open(connectionString string) (*RecordKeeper, error) {
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite db:  %s", err)
	}
	rk := &RecordKeeper{db, map[string]*sql.Stmt{}}

	err = rk.prepareSongsTable()
	if err != nil {
		return nil, fmt.Errorf("error initializing Songs table:  %s", err)
	}
	err = rk.prepareCachesTable()
	if err != nil {
		return nil, fmt.Errorf("error initialized Caches table:  %s", err)
	}

	return rk, nil
}

func (rk *RecordKeeper) Close() error {
	var err error
	for _, stmt := range rk.preparedStatementCache {
		stmtErr := stmt.Close()
		if stmtErr != nil {
			err = stmtErr
		}
	}
	closeErr := rk.DB.Close()
	if closeErr != nil {
		err = closeErr
	}
	return err
}

func (rk *RecordKeeper) Prepare(statement string) (*sql.Stmt, error) {
	if stmt, ok := rk.preparedStatementCache[statement]; ok {
		return stmt, nil
	}
	var err error
	rk.preparedStatementCache[statement], err = rk.DB.Prepare(statement)

	return rk.preparedStatementCache[statement], err
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

func (rk *RecordKeeper) prepareCachesTable() error {
	const statement = `
		CREATE TABLE IF NOT EXISTS 
		  Caches (Path TEXT NOT NULL PRIMARY KEY, Hash TEXT NOT NULL);
		CREATE INDEX IF NOT EXISTS
		  CachesHashIndex ON Caches(Hash)
    `

	_, err := rk.Exec(statement)

	if err != nil {
		return err
	}

	return nil
}

func (rk *RecordKeeper) CacheHash(path string, hash string) error {
	const statement = `
      INSERT INTO Caches(Path, Hash)
      VALUES (@Path, @Hash)
      ON CONFLICT(Path) DO UPDATE SET Hash=@Hash;
     `

	_, err := rk.Exec(statement, path, hash)
	if err != nil {
		return fmt.Errorf("error saving hash %q for file %q:  %s", hash, path, err)
	}
	return nil
}

func (rk *RecordKeeper) GetHashes() (map[string]string, error) {
	const statement = `
      SELECT Path, Hash FROM Caches
    `

	rows, err := rk.Query(statement)
	if err != nil {
		return nil, fmt.Errorf("failed to get hashes:  %s", err)
	}

	result := make(map[string]string)

	for rows.Next() {
		var path string
		var hash string
		err = rows.Scan(&path, &hash)
		if err != nil {
			return nil, fmt.Errorf("error reading cache row:  %s", err)
		}
		result[path] = hash
	}

	return result, nil
}

func (rk *RecordKeeper) RecordSongs(songs []internal.Song) error {
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
	var err error

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

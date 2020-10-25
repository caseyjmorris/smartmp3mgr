package internal

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func prepareTable(db *sql.DB) error {
	const statement = `
		CREATE TABLE IF NOT EXISTS 
		  Songs (Path TEXT NOT NULL PRIMARY KEY, Artist TEXT, Album TEXT, Title TEXT, Hash TEXT, Genre TEXT,
		  AlbumArtist TEXT, TrackNumber INTEGER, TotalTracks INTEGER, DiscNumber INTEGER, TotalDiscs INTEGER);
		CREATE INDEX IF NOT EXISTS
		  SongsHashIndex ON Songs(Hash)
    `

	_, err := db.Exec(statement)

	if err != nil {
		return err
	}

	return nil
}

func RecordSongs(db *sql.DB, songs []Song) error {
	err := prepareTable(db)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
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

	insertPrepared, err := db.Prepare(insertStatement)
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

func FetchSongs(db *sql.DB, desiredHashes []string) ([]Song, error) {
	var result []Song

	err := prepareTable(db)
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
		rows, err = db.Query(query)
	} else {
		tx, err := db.Begin()
		if err != nil {
			return result, err
		}

		_, err = db.Exec("CREATE TEMPORARY TABLE DesiredHashes(hash TEXT NOT NULL)")
		if err != nil {
			return result, err
		}

		const insertSQL = `
			INSERT INTO DesiredHashes(hash) VALUES (?)
			`

		stmt, err := db.Prepare(insertSQL)
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

		rows, err = db.Query(filteredQuery)
		if err != nil {
			return result, err
		}
	}

	defer rows.Close()
	if err != nil {
		return result, nil
	}

	for rows.Next() {
		var song Song
		err = rows.Scan(&song.Path, &song.Artist, &song.Album, &song.Title, &song.Hash, &song.Genre, &song.AlbumArtist,
			&song.TrackNumber, &song.TotalTracks, &song.DiscNumber, &song.TotalDiscs)
		if err != nil {
			return result, err
		}
		result = append(result, song)
	}

	return result, nil
}

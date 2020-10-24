package internal

import (
	"database/sql"
	"reflect"
	"testing"
)

var records = []Song{{
	Path:        "c:\\Users\\Casey\\Song1.mp3",
	Artist:      "Starpoint",
	Album:       "Restless",
	Title:       "Object of My Desire",
	Hash:        "adfd",
	Genre:       "R&B",
	AlbumArtist: "Starpoint",
	TrackNumber: 1,
	TotalTracks: 12,
	DiscNumber:  1,
	TotalDiscs:  1,
}, {
	Path:        "c:\\Users\\Casey\\Song2.mp3",
	Artist:      "浜崎あゆみ",
	Album:       "オリアの木",
	Title:       "オリアの木",
	Hash:        "cfdkfkslafj",
	Genre:       "J-Pop",
	AlbumArtist: "浜崎あゆみ",
	TrackNumber: 2,
	TotalTracks: 2,
	DiscNumber:  1,
	TotalDiscs:  2,
}}

const connectionString = "file:test.db?cache=shared&mode=memory"

func TestRecordAndFetchSongs(t *testing.T) {
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	err = RecordSongs(db, records)
	if err != nil {
		t.Error(err)
	}
	result, err := FetchSongs(db, []string{})
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(records, result) {
		t.Errorf("Records returned don't match.  \r\nExpected:  %v  \r\nActual:  %v", records, result)
	}
}

func TestFilteredFetchSongs(t *testing.T) {
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	err = RecordSongs(db, records)
	if err != nil {
		t.Error(err)
	}
	result, err := FetchSongs(db, []string{"cfdkfkslafj"})
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(records[1:], result) {
		t.Errorf("Records returned don't match.  \r\nExpected:  %v  \r\nActual:  %v", records[1:], result)
	}
}

func TestInsertIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	err = RecordSongs(db, records)
	if err != nil {
		t.Error(err)
	}
	err = RecordSongs(db, records)
	if err != nil {
		t.Error(err)
	}
	result, err := FetchSongs(db, []string{})
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(records, result) {
		t.Errorf("Records returned don't match.  \r\nExpected:  %v  \r\nActual:  %v", records, result)
	}
}

package records

import (
	"github.com/caseyjmorris/smartmp3mgr/mp3util"
	"reflect"
	"testing"
)

var records = []mp3util.Song{{
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
	db, err := Open(connectionString)
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	tx, _ := db.Begin()
	for _, record := range records {
		err = db.RecordSong(record)
		if err != nil {
			t.Error(err)
		}
	}
	_ = tx.Commit()

	result, err := db.FetchSongs()
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(records, result) {
		t.Errorf("Records returned don't match.  \r\nExpected:  %v  \r\nActual:  %v", records, result)
	}
}

func TestInsertIdempotent(t *testing.T) {
	db, err := Open(connectionString)
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	tx, _ := db.Begin()
	for _, record := range records {
		err = db.RecordSong(record)
		if err != nil {
			t.Error(err)
		}
	}
	_ = tx.Commit()
	tx, _ = db.Begin()
	for _, record := range records {
		err = db.RecordSong(record)
		if err != nil {
			t.Error(err)
		}
	}
	_ = tx.Commit()

	result, err := db.FetchSongs()
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(records, result) {
		t.Errorf("Records returned don't match.  \r\nExpected:  %v  \r\nActual:  %v", records, result)
	}
}

func TestCacheFunctionality(t *testing.T) {
	db, _ := Open(connectionString)
	_ = db.CacheHash("ABC", "123")
	_ = db.CacheHash("DEF", "456")
	_ = db.CacheHash("ABC", "789")
	result, _ := db.GetHashes()
	expected := make(map[string]string)
	expected["DEF"] = "456"
	expected["ABC"] = "789"

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Records returned don't match.  \r\nExpected:  %v  \r\nActual:  %v", records, result)
	}
}

package records

import (
	"github.com/caseyjmorris/smartmp3mgr/mp3"
	"reflect"
	"testing"
)

var records = []mp3.Song{{
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

	err = db.RecordSongs(records)
	if err != nil {
		t.Error(err)
	}
	result, err := db.FetchSongs([]string{})
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(records, result) {
		t.Errorf("Records returned don't match.  \r\nExpected:  %v  \r\nActual:  %v", records, result)
	}
}

func TestFilteredFetchSongs(t *testing.T) {
	db, err := Open(connectionString)
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	err = db.RecordSongs(records)
	if err != nil {
		t.Error(err)
	}
	result, err := db.FetchSongs([]string{"cfdkfkslafj"})
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(records[1:], result) {
		t.Errorf("Records returned don't match.  \r\nExpected:  %v  \r\nActual:  %v", records[1:], result)
	}
}

func TestInsertIdempotent(t *testing.T) {
	db, err := Open(connectionString)
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	err = db.RecordSongs(records)
	if err != nil {
		t.Error(err)
	}
	err = db.RecordSongs(records)
	if err != nil {
		t.Error(err)
	}
	result, err := db.FetchSongs([]string{})
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

package mp3util

type Song struct {
	Path, Artist, Album, Title, Hash, Genre, AlbumArtist string
	TrackNumber, TotalTracks, DiscNumber, TotalDiscs     int
}

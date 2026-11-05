package main

import (
	"path/filepath"
	"time"
)

// fav is a series the viewer saved, with what a grid needs to show it
// without asking the server again. It keeps the episodes rather than the
// line its grid showed, which may be a chart's score or a day's heat and
// would be stale on the 收藏 page.
type fav struct {
	SeriesID string    `json:"series_id"`
	Title    string    `json:"title"`
	Cover    string    `json:"cover,omitempty"`
	Episodes int       `json:"episodes,omitempty"`
	Added    time.Time `json:"added"`
}

// favorites keeps the series saved, the latest first, in favorites.json.
// It belongs to the main thread.
type favorites struct {
	store *store[fav]
	list  []fav
	ids   map[string]bool
	rev   int // counts the changes, so the grid knows when to rebuild
}

func loadFavorites(dir string) *favorites {
	f := &favorites{ids: map[string]bool{}}
	f.store, f.list = openStore[fav](filepath.Join(dir, "favorites.json"))
	for _, x := range f.list {
		f.ids[x.SeriesID] = true
	}
	return f
}

func (f *favorites) has(id string) bool { return f.ids[id] }

// toggle saves the series, or drops it when it is saved already, and
// reports whether it is saved now.
func (f *favorites) toggle(x fav) bool {
	if f.has(x.SeriesID) {
		f.remove(x.SeriesID)
		return false
	}
	f.add(x)
	return true
}

// add saves the series, newest first; saving one twice only moves it to
// the front.
func (f *favorites) add(x fav) {
	if f.has(x.SeriesID) || x.SeriesID == "" {
		return
	}
	x.Added = time.Now()
	f.list = append([]fav{x}, f.list...)
	f.ids[x.SeriesID] = true
	f.changed()
}

func (f *favorites) remove(id string) {
	if !f.has(id) {
		return
	}
	list := make([]fav, 0, len(f.list)-1)
	for _, x := range f.list {
		if x.SeriesID != id {
			list = append(list, x)
		}
	}
	f.list = list
	delete(f.ids, id)
	f.changed()
}

func (f *favorites) changed() {
	f.rev++
	f.store.put(f.list)
}

// cards is the saved series as a grid shows them, the latest first.
func (f *favorites) cards() []card {
	cards := make([]card, 0, len(f.list))
	for _, x := range f.list {
		cards = append(cards, card{ID: x.SeriesID, Title: x.Title, Cover: x.Cover,
			Sub: episodeCount(x.Episodes), Episodes: x.Episodes})
	}
	return cards
}

// favOf is the series as it would be saved.
func favOf(k card) fav {
	return fav{SeriesID: k.ID, Title: k.Title, Cover: k.Cover, Episodes: k.Episodes}
}

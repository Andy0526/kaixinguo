package main

import (
	"path/filepath"
	"time"
)

// watch is where the viewer is in one series.
type watch struct {
	SeriesID string    `json:"series_id"`
	Title    string    `json:"title"`
	Cover    string    `json:"cover,omitempty"`
	Episode  int       `json:"episode"`
	Position float64   `json:"position,omitempty"` // seconds into the episode
	Duration float64   `json:"duration,omitempty"`
	Total    int       `json:"total,omitempty"` // episodes in the series
	Finished bool      `json:"finished,omitempty"`
	Updated  time.Time `json:"updated"`
}

// historyMax is how many series the history keeps.
const historyMax = 200

// history keeps the series watched, the latest first, in history.json. It
// belongs to the main thread.
type history struct {
	store *store[watch]
	list  []watch
}

func loadHistory(dir string) *history {
	h := &history{}
	h.store, h.list = openStore[watch](filepath.Join(dir, "history.json"))
	return h
}

func (h *history) get(id string) (watch, bool) {
	for _, w := range h.list {
		if w.SeriesID == id {
			return w, true
		}
	}
	return watch{}, false
}

// record moves w to the front.
func (h *history) record(w watch) {
	w.Updated = time.Now()
	list := []watch{w}
	for _, o := range h.list {
		if o.SeriesID != w.SeriesID {
			list = append(list, o)
		}
	}
	if len(list) > historyMax {
		list = list[:historyMax]
	}
	h.list = list
	h.store.put(list)
}

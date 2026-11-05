package main

import (
	"encoding/json"
	"os"
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

// history keeps the series watched, the latest first, in a file. It
// belongs to the main thread; it writes the file on a goroutine.
type history struct {
	path string
	list []watch
	save chan []watch
}

func loadHistory(dir string) *history {
	h := &history{path: filepath.Join(dir, "history.json"), save: make(chan []watch, 1)}
	if data, err := os.ReadFile(h.path); err == nil {
		_ = json.Unmarshal(data, &h.list)
	}
	go h.writer()
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
	if len(list) > 200 {
		list = list[:200]
	}
	h.list = list
	snapshot := append([]watch(nil), list...)
	// Only the latest snapshot matters.
	select {
	case <-h.save:
	default:
	}
	h.save <- snapshot
}

func (h *history) writer() {
	for list := range h.save {
		data, err := json.Marshal(list)
		if err != nil {
			continue
		}
		tmp := h.path + ".tmp"
		if os.WriteFile(tmp, data, 0o644) == nil {
			_ = os.Rename(tmp, h.path)
		}
	}
}

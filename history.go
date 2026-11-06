package main

import (
	"path/filepath"
	"time"
)

// episodeWatch is how far the viewer got into one episode.
type episodeWatch struct {
	Position float64 `json:"position,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	Finished bool    `json:"finished,omitempty"`
}

// watch is where the viewer is in one series.
type watch struct {
	SeriesID string               `json:"series_id"`
	Title    string               `json:"title"`
	Cover    string               `json:"cover,omitempty"`
	Episode  int                  `json:"episode"`
	Position float64              `json:"position,omitempty"` // seconds into the episode
	Season   int                  `json:"season,omitempty"`   // the season's number, when it has several
	Duration float64              `json:"duration,omitempty"`
	Total    int                  `json:"total,omitempty"` // episodes in the series
	Finished bool                 `json:"finished,omitempty"`
	Done     bool                 `json:"done,omitempty"` // the whole series
	Want     bool                 `json:"want,omitempty"`
	Episodes map[int]episodeWatch `json:"episodes,omitempty"`
	Updated  time.Time            `json:"updated"`
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
	changed := false
	for i, w := range h.list {
		if w.Episodes == nil && w.Episode > 0 {
			w.Episodes = map[int]episodeWatch{w.Episode: {
				Position: w.Position, Duration: w.Duration, Finished: w.Finished,
			}}
			changed = true
		}
		if w.Finished && w.Total > 0 && w.Episode >= w.Total && !w.Done {
			w.Done = true
			changed = true
		}
		if w.Season == 0 && w.Episode > 0 {
			if n := seasonNumber(w.Title); n > 0 {
				w.Season = n
				changed = true
			}
		}
		h.list[i] = w
	}
	if changed {
		h.store.put(h.list)
	}
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

func (h *history) remove(id string) {
	list := make([]watch, 0, len(h.list))
	for _, w := range h.list {
		if w.SeriesID != id {
			list = append(list, w)
		}
	}
	h.list = list
	h.store.put(list)
}

// episode returns what was watched of episode n.
func (w watch) episode(n int) (episodeWatch, bool) {
	e, ok := w.Episodes[n]
	return e, ok
}

// episodeState reports whether episode n was watched and, if it is
// partly watched, how far.
func (w watch) episodeState(n int) (bool, float64) {
	e, ok := w.episode(n)
	if !ok {
		return false, 0
	}
	if e.Finished {
		return true, 1
	}
	if e.Duration > 0 {
		p := e.Position / e.Duration
		if p < 0 {
			p = 0
		} else if p > 1 {
			p = 1
		}
		return false, p
	}
	return false, 0
}

// episodeStart is where to start episode n again: the saved second, or 0
// for an episode not started or watched through.
func (w watch) episodeStart(n int) float64 {
	e, ok := w.episode(n)
	if !ok || e.Finished {
		return 0
	}
	return e.Position
}

// status is the filter the recent list puts a series under.
func (w watch) status() string {
	switch {
	case w.Done:
		return "done"
	case w.Episode > 0:
		return "watching"
	default:
		return "want"
	}
}

// filtered returns the history under a recent filter: "", want, watching
// or done.
func (h *history) filtered(status string) []watch {
	if status == "" {
		return h.list
	}
	out := make([]watch, 0, len(h.list))
	for _, w := range h.list {
		if w.status() == status {
			out = append(out, w)
		}
	}
	return out
}

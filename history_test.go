package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// An episode records where it stopped, then a full watch, and the last
// episode ends the series.
func TestEpisodeProgress(t *testing.T) {
	a := testApp(t)
	sp := a.series["1"]
	a.recordWatch(sp, 1, 30, 100, false)
	w, ok := a.hist.get("1")
	if !ok {
		t.Fatal("no history")
	}
	if done, frac := w.episodeState(1); done || frac != 0.3 {
		t.Fatalf("episode 1 done=%v frac=%v", done, frac)
	}
	if start := w.episodeStart(1); start != 30 {
		t.Fatalf("resume at %v, want 30", start)
	}
	if w.status() != "watching" {
		t.Fatalf("status %q, want watching", w.status())
	}
	if w.Want {
		t.Fatal("watching a series still wants it")
	}

	a.recordWatch(sp, 1, 0, 0, true)
	w, _ = a.hist.get("1")
	if done, frac := w.episodeState(1); !done || frac != 1 {
		t.Fatalf("episode 1 done=%v frac=%v after it ended", done, frac)
	}
	if start := w.episodeStart(1); start != 0 {
		t.Fatalf("a watched episode starts at %v, want 0", start)
	}
	if w.Done {
		t.Fatal("one episode ended a 130-episode series")
	}

	a.recordWatch(sp, 130, 0, 0, true)
	w, _ = a.hist.get("1")
	if !w.Done || w.status() != "done" {
		t.Fatalf("last episode done=%v status=%q", w.Done, w.status())
	}
}

// A series watched through every episode is done even if the last one
// was not the one that ended last.
func TestAllEpisodesDone(t *testing.T) {
	a := testApp(t)
	sp := a.series["1"]
	sp.episodes = sp.episodes[:2]
	sp.series.EpisodeCount = 2
	a.recordWatch(sp, 1, 0, 0, true)
	a.recordWatch(sp, 2, 0, 0, true)
	if w, _ := a.hist.get("1"); !w.Done {
		t.Fatal("both episodes watched, series not done")
	}
}

// The recent list filters want, watching and done.
func TestRecentFilters(t *testing.T) {
	a := testApp(t)
	a.hist.record(watch{SeriesID: "w", Title: "想看的剧", Want: true})
	a.hist.record(watch{SeriesID: "v", Title: "在看的剧", Episode: 4})
	a.hist.record(watch{SeriesID: "d", Title: "看完的剧", Episode: 3, Done: true})
	tt := ui.NewTester(a.view, 1180, 760)
	tt.Advance(1e9)
	for _, s := range []string{"全部", "想看", "正在看", "已看完", "想看的剧", "在看的剧", "看完的剧"} {
		if !tt.HasText(s) {
			t.Fatalf("no %q in %q", s, tt.Texts())
		}
	}
	if err := tt.Click("已看完"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(1e9)
	if !tt.HasText("看完的剧") || tt.HasText("想看的剧") || tt.HasText("在看的剧") {
		t.Fatalf("done filter shows %q", tt.Texts())
	}
	if err := tt.Click("想看"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(1e9)
	if !tt.HasText("想看的剧") || tt.HasText("看完的剧") {
		t.Fatalf("want filter shows %q", tt.Texts())
	}
	if err := tt.Click("正在看"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(1e9)
	if !tt.HasText("在看的剧") || tt.HasText("想看的剧") {
		t.Fatalf("watching filter shows %q", tt.Texts())
	}
	if err := tt.Click("全部"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(1e9)
	for _, s := range []string{"想看的剧", "在看的剧", "看完的剧"} {
		if !tt.HasText(s) {
			t.Fatalf("all filter lost %q: %q", s, tt.Texts())
		}
	}
}

// The series page marks a want, and the recent list shows it.
func TestWantButton(t *testing.T) {
	a := testApp(t)
	a.router.Push("/series/1")
	tt := ui.NewTester(a.view, 1180, 760)
	tt.Advance(1e9)
	if !tt.HasText("想看") {
		t.Fatalf("no want button: %q", tt.Texts())
	}
	if err := tt.Click("想看"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(1e9)
	w, ok := a.hist.get("1")
	if !ok || !w.Want || w.status() != "want" {
		t.Fatalf("history %+v", w)
	}
	if !tt.HasText("已想看") || !tt.HasText("播放第 1 集") {
		t.Fatalf("want series has no play button: %q", tt.Texts())
	}
	if err := tt.Click("已想看"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(1e9)
	if _, ok := a.hist.get("1"); ok {
		t.Fatal("the want stays after a second click")
	}
}

// History from before per-episode progress still resumes.
func TestEpisodeProgressKeepsOldHistory(t *testing.T) {
	a := testApp(t)
	a.hist.record(watch{SeriesID: "1", Title: "浅浅归聿", Episode: 7, Position: 42, Updated: time.Now()})
	w, _ := a.hist.get("1")
	if start := w.episodeStart(7); start != 0 {
		t.Fatalf("old history starts at %v, want 0", start)
	}
	id, n, start, _ := a.resumePoint(a.series["1"])
	if id != "1" || n != 7 || start != 39 {
		t.Fatalf("old history resumes at %s %d %v", id, n, start)
	}
}

func TestOldHistoryMigrates(t *testing.T) {
	dir := t.TempDir()
	old, _ := json.Marshal([]watch{{
		SeriesID: "1", Title: "浅浅归聿", Episode: 130, Position: 42,
		Total: 130, Finished: true,
	}})
	if err := os.WriteFile(filepath.Join(dir, "history.json"), old, 0o644); err != nil {
		t.Fatal(err)
	}
	h := loadHistory(dir)
	defer h.store.flush()
	w, ok := h.get("1")
	if !ok {
		t.Fatal("no history")
	}
	if done, _ := w.episodeState(130); !done {
		t.Fatal("the old current episode is not marked watched")
	}
	if !w.Done {
		t.Fatal("an old finished last episode does not finish the series")
	}
}

func TestEpisodeProgressRoundTrip(t *testing.T) {
	dir := t.TempDir()
	h := loadHistory(dir)
	h.record(watch{
		SeriesID: "1", Title: "浅浅归聿", Episode: 2, Total: 130,
		Episodes: map[int]episodeWatch{
			1: {Finished: true},
			2: {Position: 30, Duration: 100},
		},
	})
	h.store.flush()
	h2 := loadHistory(dir)
	defer h2.store.flush()
	w, ok := h2.get("1")
	if !ok {
		t.Fatal("no history after a round trip")
	}
	if done, _ := w.episodeState(1); !done {
		t.Fatal("episode 1 lost its watched mark")
	}
	if done, frac := w.episodeState(2); done || frac != 0.3 {
		t.Fatalf("episode 2 done=%v frac=%v", done, frac)
	}
}

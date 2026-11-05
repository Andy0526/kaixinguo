package main

import (
	"strings"
	"testing"

	"github.com/beilunyang/openhongguo/sdk/model"
	"github.com/egoist/mygo/ui"
)

// testApp is an app with a catalog already loaded, which needs no network.
func testApp(t *testing.T) *app {
	a := newApp(t.TempDir(), t.TempDir())
	// Cleanups run last first: the files are written before the
	// directories are removed.
	t.Cleanup(func() {
		a.hist.store.flush()
		a.favs.store.flush()
	})
	a.home = newFeed(nil)
	a.home.add(page{cards: []card{
		{ID: "1", Title: "浅浅归聿", Sub: "124 集", Episodes: 124},
		{ID: "2", Title: "大唐第一军火商", Sub: "521 集", Episodes: 521},
	}})
	a.home.started, a.home.more = true, false
	eps := []model.Episode{}
	for i := 1; i <= 130; i++ {
		eps = append(eps, model.Episode{ID: "e" + itoa(i), SeriesID: "1", Number: i, Playable: i != 3})
	}
	a.series["1"] = &seriesPage{id: "1", series: &model.Series{
		SeriesSummary: model.SeriesSummary{ID: "1", Title: "浅浅归聿", Description: "绝症少女穿进财阀甜宠小说。", EpisodeCount: 130},
		Status:        model.SeriesEnd, Platform: model.PlatformOuter, Heat: 41658011,
	}, episodes: eps}
	return a
}

// The home page shows the recommendations, and a click on one opens its
// page, with its episodes in hundreds.
func TestHomeToSeries(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1180, 760)
	for _, s := range []string{"推荐", "排行榜", "新剧上线", "浅浅归聿", "大唐第一军火商"} {
		if !tt.HasText(s) {
			t.Fatalf("no %q in %q", s, tt.Texts())
		}
	}
	if err := tt.Click("浅浅归聿"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(1e9)
	if got := a.router.Path(); got != "/series/1" {
		t.Fatalf("path %q after a click", got)
	}
	for _, s := range []string{"选集", "共 130 集", "播放第 1 集", "全 130 集  ·  热度 4165万", "1–100", "101–130"} {
		if !tt.HasText(s) {
			t.Fatalf("no %q in %q", s, tt.Texts())
		}
	}
	if hasExact(tt, "101") {
		t.Fatal("the second hundred shows before it is chosen")
	}
	if err := tt.Click("101–130"); err != nil {
		t.Fatal(err)
	}
	if !hasExact(tt, "130") || hasExact(tt, "100") {
		t.Fatalf("the second hundred does not show: %q", tt.Texts())
	}
}

// What the viewer watched shows in the sidebar, and the series offers to go
// on from there.
func TestResume(t *testing.T) {
	a := testApp(t)
	a.hist.record(watch{SeriesID: "1", Title: "浅浅归聿", Episode: 7, Position: 42})
	a.router.Push("/series/1")
	tt := ui.NewTester(a.view, 1180, 760)
	tt.Advance(1e9)
	if !tt.HasText("最近观看") || !tt.HasText("继续看第 7 集") {
		t.Fatalf("texts %q", tt.Texts())
	}
	id, n, start, _ := a.resumePoint(a.series["1"])
	if id != "1" || n != 7 || start != 39 {
		t.Fatalf("resumes at episode %d, %v s", n, start)
	}
}

// hasExact reports whether the last frame shows the text s, whole.
func hasExact(tt *ui.Tester, s string) bool {
	for _, x := range tt.Texts() {
		if x == s {
			return true
		}
	}
	return false
}

func TestDayLabel(t *testing.T) {
	for d, want := range map[string]string{"20261105": "今天", "20261106": "明天", "20261104": "昨天", "20261112": "11月12日"} {
		if got := dayLabel(d, "20261105"); got != want {
			t.Errorf("dayLabel(%s) = %q, want %q", d, got, want)
		}
	}
}

func TestParseRange(t *testing.T) {
	for _, c := range []struct {
		h          string
		start, end int64
		partial    bool
		ok         bool
	}{
		{"", 0, 999, false, true},
		{"bytes=0-1", 0, 1, true, true},
		{"bytes=100-", 100, 999, true, true},
		{"bytes=900-5000", 900, 999, true, true},
		{"bytes=-10", 990, 999, true, true},
		{"bytes=1000-", 0, 0, false, false},
		{"bytes=0-1,5-6", 0, 0, false, false},
	} {
		s, e, p, ok := parseRange(c.h, 1000)
		if ok != c.ok || ok && (s != c.start || e != c.end || p != c.partial) {
			t.Errorf("parseRange(%q) = %d %d %v %v", c.h, s, e, p, ok)
		}
	}
}

// Fresh recommendations go before the cached ones, which stay, so the grid
// never shrinks as the app refreshes it.
func TestStaleFeedMerges(t *testing.T) {
	f := newFeed(nil)
	f.add(page{cards: []card{{ID: "a"}, {ID: "b"}, {ID: "c"}}})
	f.stale = true
	f.add(page{cards: []card{{ID: "d"}, {ID: "b"}}, cursor: "next", more: true})
	var ids []string
	for _, k := range f.cards {
		ids = append(ids, k.ID)
	}
	if got := strings.Join(ids, ","); got != "d,b,a,c" || f.cursor != "next" || f.stale {
		t.Fatalf("cards %s, cursor %q, stale %v", got, f.cursor, f.stale)
	}
}

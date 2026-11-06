package main

import (
	"strings"
	"testing"

	"github.com/beilunyang/openhongguo/sdk/model"
	"github.com/egoist/mygo/ui"
)

func TestChineseNumber(t *testing.T) {
	for s, want := range map[string]int{"一": 1, "三": 3, "十": 10, "十一": 11, "二十": 20, "二十三": 23, "两": 2, "12": 12, "一百零五": 105} {
		if got := chineseNumber(s); got != want {
			t.Errorf("chineseNumber(%q) = %d, want %d", s, got, want)
		}
	}
}

// Search lists the seasons of a series in their order, where the first of
// them came, and leaves the rest alone; another series of the same name
// without the colon keeps to itself.
func TestOrderSeasons(t *testing.T) {
	titles := []string{"持械入宋：第一季", "持械入宋：第十季", "一品布衣", "持械入宋：第二季", "持械入宋第五季", "持械入宋：第十一季", "持械入宋第三季"}
	var cards []card
	for i, s := range titles {
		cards = append(cards, card{ID: itoa(i), Title: s})
	}
	orderSeasons(cards)
	var got []string
	for _, k := range cards {
		got = append(got, k.Title)
	}
	want := "持械入宋：第一季,持械入宋：第二季,持械入宋：第十季,持械入宋：第十一季,一品布衣,持械入宋第三季,持械入宋第五季"
	if strings.Join(got, ",") != want {
		t.Fatalf("order %s", strings.Join(got, ","))
	}
}

// A search page with nothing new ends the feed, rather than the grid
// asking for the same page forever.
func TestRepeatedPageEndsFeed(t *testing.T) {
	f := newFeed(nil)
	f.add(page{cards: []card{{ID: "a"}, {ID: "b"}}, cursor: "20", more: true})
	f.add(page{cards: []card{{ID: "b"}, {ID: "a"}}, cursor: "40", more: true})
	if f.more || len(f.cards) != 2 {
		t.Fatalf("more %v, %d cards", f.more, len(f.cards))
	}
}

// seasonsApp has a series in three seasons, the last still to come.
func seasonsApp(t *testing.T) *app {
	a := testApp(t)
	seasons := []model.Season{
		{SeriesID: "1", Index: 1, EpisodeCount: 130},
		{SeriesID: "s2", Index: 2, EpisodeCount: 80},
		{SeriesID: "s3", Index: 3},
	}
	a.series["1"].seasons = seasons
	a.series["1"].series.IsMultiSeason = true
	a.series["s2"] = &seriesPage{id: "s2", seasons: seasons, series: &model.Series{
		SeriesSummary: model.SeriesSummary{ID: "s2", Title: "浅浅归聿第二季", EpisodeCount: 80},
		Platform:      model.PlatformOuter, IsMultiSeason: true,
	}, episodes: []model.Episode{{ID: "s2e1", SeriesID: "s2", Number: 1, Playable: true}}}
	return a
}

// The series page offers its other seasons, and choosing one shows it in
// place of this one.
func TestSeasonPicker(t *testing.T) {
	a := seasonsApp(t)
	a.router.Push("/series/1")
	tt := ui.NewTester(a.view, 1180, 760)
	tt.Advance(1e9)
	for _, s := range []string{"第 1 季", "第 2 季", "第 3 季"} {
		if !tt.HasText(s) {
			t.Fatalf("no %q in %q", s, tt.Texts())
		}
	}
	if err := tt.Click("第 2 季"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(1e9)
	if got := a.router.Path(); got != "/series/s2" || !tt.HasText("浅浅归聿第二季") {
		t.Fatalf("path %q, texts %q", got, tt.Texts())
	}
	a.router.Back()
	if got := a.router.Path(); got != "/discover" {
		t.Fatalf("Back went to %q, not out of the series", got)
	}
}

// After the last episode of a season comes the next season's first, and a
// season watched to its end offers the next.
func TestNextSeason(t *testing.T) {
	a := seasonsApp(t)
	sp := a.series["1"]
	if id, n, ok := a.following(sp, 5); !ok || id != "1" || n != 6 {
		t.Fatalf("after episode 5: %s %d %v", id, n, ok)
	}
	if id, n, ok := a.following(sp, 130); !ok || id != "s2" || n != 1 {
		t.Fatalf("after the last episode: %s %d %v", id, n, ok)
	}
	// The third season has no episodes yet.
	if _, _, ok := a.following(a.series["s2"], 1); ok {
		t.Fatal("went on to a season still to come")
	}
	a.hist.record(watch{SeriesID: "1", Episode: 130, Finished: true})
	if id, n, _, label := a.resumePoint(sp); id != "s2" || n != 1 || label != "播放第 2 季" {
		t.Fatalf("resumes at %s %d, %q", id, n, label)
	}
}

// Seasons the server sends without numbers take them from their titles,
// the first season being the one title without a number; seasons with
// no telling keep their order.
func TestNumberSeasons(t *testing.T) {
	label := func(ss []model.Season) string {
		var out []string
		for _, s := range ss {
			out = append(out, itoa(s.Index)+":"+s.Title)
		}
		return strings.Join(out, ",")
	}
	for _, c := range []struct {
		in   []model.Season
		want string
	}{
		{ // numbers from the server stand
			[]model.Season{{Index: 2, Title: "b"}, {Index: 1, Title: "a"}},
			"1:a,2:b",
		},
		{ // from the titles
			[]model.Season{{Title: "持械入宋：第十一季"}, {Title: "持械入宋：第一季"}, {Title: "持械入宋：第二季"}},
			"1:持械入宋：第一季,2:持械入宋：第二季,11:持械入宋：第十一季",
		},
		{ // the first season's title has no number
			[]model.Season{{Title: "浅浅归聿第三季"}, {Title: "浅浅归聿"}, {Title: "浅浅归聿第二季"}},
			"1:浅浅归聿,2:浅浅归聿第二季,3:浅浅归聿第三季",
		},
		{ // nothing to tell: the server's order
			[]model.Season{{Title: "甲"}, {Title: "乙"}, {Title: "丙"}},
			"1:甲,2:乙,3:丙",
		},
	} {
		if got := label(numberSeasons(c.in)); got != c.want {
			t.Errorf("got %s, want %s", got, c.want)
		}
	}
}

// Playing a series in several seasons keeps the season on its history, so
// the favorites page can say which one is under way.
func TestRecordSeason(t *testing.T) {
	a := seasonsApp(t)
	sp := a.series["s2"]
	a.recordWatch(sp, 5, 20, 40, false)
	w, ok := a.hist.get("s2")
	if !ok || w.Season != 2 {
		t.Fatalf("watch %+v, want season 2", w)
	}
	if got := seasonNumber("持械入宋：第三季"); got != 3 {
		t.Fatalf("seasonNumber = %d, want 3", got)
	}
	if got := seasonNumber("浅浅归聿"); got != 0 {
		t.Fatalf("seasonNumber of a single season = %d, want 0", got)
	}
}

package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// Saving a series keeps it, newest first, and the file it is kept in is
// what the next launch reads.
func TestFavoritesPersist(t *testing.T) {
	dir := t.TempDir()
	f := loadFavorites(dir)
	t.Cleanup(f.store.flush)
	if f.has("1") {
		t.Fatal("a fresh list has a series in it")
	}
	if !f.toggle(fav{SeriesID: "1", Title: "浅浅归聿"}) {
		t.Fatal("toggle did not save the series")
	}
	f.add(fav{SeriesID: "2", Title: "大唐第一军火商"})
	f.add(fav{SeriesID: "2", Title: "大唐第一军火商"}) // saving twice saves once
	if got := favIDs(f); got != "2,1" {
		t.Fatalf("saved %q, want the newest first", got)
	}

	if got := savedIDs(t, f); got != "2,1" {
		t.Fatalf("favorites.json holds %q", got)
	}
	again := loadFavorites(dir)
	t.Cleanup(again.store.flush)
	if got := favIDs(again); got != "2,1" {
		t.Fatalf("read back %q", got)
	}
	if !again.has("1") || again.has("3") {
		t.Fatal("the list read back does not know what is in it")
	}
	if again.toggle(fav{SeriesID: "1"}) {
		t.Fatal("toggle saved a series that was saved already")
	}
	if got := savedIDs(t, again); got != "2" {
		t.Fatalf("after dropping one, favorites.json holds %q", got)
	}
}

// Removing a series that was never saved changes nothing.
func TestFavoritesRemoveUnknown(t *testing.T) {
	f := loadFavorites(t.TempDir())
	t.Cleanup(f.store.flush)
	f.add(fav{SeriesID: "1"})
	rev := f.rev
	f.remove("nope")
	if f.rev != rev || favIDs(f) != "1" {
		t.Fatalf("rev %d→%d, list %q", rev, f.rev, favIDs(f))
	}
}

// A series saved from a chart keeps its episodes, not the chart's score,
// which would be stale on the 收藏 page; one whose episodes no page told
// shows no line at all.
func TestFavoriteKeepsEpisodes(t *testing.T) {
	f := loadFavorites(t.TempDir())
	t.Cleanup(f.store.flush)
	f.add(favOf(card{ID: "3", Title: "红英卤味摊", Sub: "579万热搜值", Rank: 1, Episodes: 80}))
	f.add(favOf(card{ID: "4", Title: "明天上线的剧", Sub: "明天 20:00 上线"}))
	cards := f.cards()
	if len(cards) != 2 || cards[0].Sub != "" || cards[1].Sub != "80 集" || cards[1].Rank != 0 {
		t.Fatalf("cards %+v", cards)
	}
}

// The 收藏 button on a series page saves it; the sidebar counts it, the
// 收藏 page lists it, and the card's menu takes it off again.
func TestFavoriteFromSeriesPage(t *testing.T) {
	a := testApp(t)
	a.router.Push("/series/1")
	tt := ui.NewTester(a.view, 1180, 760)
	tt.Advance(time.Second)
	if !tt.HasText("收藏") {
		t.Fatalf("no 收藏 on the series page: %q", tt.Texts())
	}
	if err := tt.Click("收藏这部剧"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.favs.has("1") {
		t.Fatal("the click did not save the series")
	}
	if !tt.HasText("已收藏") {
		t.Fatalf("the button does not show it is saved: %q", tt.Texts())
	}

	a.router.Push("/favorites")
	tt.Advance(time.Second)
	for _, s := range []string{"浅浅归聿", "130 集"} {
		if !hasExact(tt, s) {
			t.Fatalf("no %q on the 收藏 page: %q", s, tt.Texts())
		}
	}

	// The same card's menu drops it, and the page goes back to empty.
	if err := tt.RightClick("浅浅归聿"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(tt.Menu(), ","); got != "取消收藏" {
		t.Fatalf("the card's menu is %q", got)
	}
	if err := tt.ChooseMenuItem("取消收藏"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if a.favs.has("1") {
		t.Fatal("the series is still saved")
	}
	if !tt.HasText("还没有收藏") {
		t.Fatalf("the empty 收藏 page says %q", tt.Texts())
	}
}

// A series saved from a grid's menu shows on the 收藏 page, with the line
// under its title that the grid showed.
func TestFavoriteFromGridMenu(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.view, 1180, 760)
	tt.Advance(time.Second)
	if err := tt.RightClick("大唐第一军火商"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("收藏"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if got := favIDs(a.favs); got != "2" {
		t.Fatalf("saved %q", got)
	}
	a.router.Push("/favorites")
	tt.Advance(time.Second)
	for _, s := range []string{"收藏", "大唐第一军火商", "521 集"} {
		if !tt.HasText(s) {
			t.Fatalf("no %q on the 收藏 page: %q", s, tt.Texts())
		}
	}
}

func favIDs(f *favorites) string {
	var ids []string
	for _, x := range f.list {
		ids = append(ids, x.SeriesID)
	}
	return strings.Join(ids, ",")
}

// savedIDs is the series in the file of f, once what f put is written.
func savedIDs(t *testing.T, f *favorites) string {
	t.Helper()
	f.store.flush()
	data, err := os.ReadFile(f.store.path)
	if err != nil {
		t.Fatal(err)
	}
	var list []fav
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, x := range list {
		ids = append(ids, x.SeriesID)
	}
	return strings.Join(ids, ",")
}

// A saved series shows how far the viewer got into it, on its poster.
func TestFavoriteShowsWatchProgress(t *testing.T) {
	a := testApp(t)
	a.favs.add(favOf(card{ID: "1", Title: "浅浅归聿", Episodes: 130}))
	a.hist.record(watch{SeriesID: "1", Title: "浅浅归聿", Episode: 7, Total: 130,
		Episodes: map[int]episodeWatch{7: {Position: 20, Duration: 40}}})
	a.router.Push("/favorites")
	tt := ui.NewTester(a.view, 1180, 760)
	tt.Advance(time.Second)
	if !tt.HasText("看到第 7 集") {
		t.Fatalf("no watch progress on the 收藏 page: %q", tt.Texts())
	}
}

// A series in several seasons says which season and episode are under way.
func TestFavoriteShowsSeasonProgress(t *testing.T) {
	a := testApp(t)
	a.favs.add(fav{SeriesID: "s2", Title: "浅浅归聿第二季", Episodes: 80})
	a.hist.record(watch{SeriesID: "s2", Title: "浅浅归聿第二季", Episode: 5, Total: 80,
		Episodes: map[int]episodeWatch{5: {Position: 10, Duration: 40}}})
	a.router.Push("/favorites")
	tt := ui.NewTester(a.view, 1180, 760)
	tt.Advance(time.Second)
	if !tt.HasText("第 2 季 · 第 5 集") {
		t.Fatalf("no season progress on the 收藏 page: %q", tt.Texts())
	}
}

// The fraction counts finished episodes and the part of the current one.
func TestWatchFraction(t *testing.T) {
	w := watch{Episode: 3, Total: 4, Episodes: map[int]episodeWatch{
		1: {Finished: true}, 2: {Finished: true}, 3: {Position: 20, Duration: 40},
	}}
	if got := watchFraction(w, 4); got != 2.5/4 {
		t.Fatalf("progress %v, want 2.5/4", got)
	}
	w.Done = true
	if got := watchFraction(w, 4); got != 1 {
		t.Fatalf("finished progress %v, want 1", got)
	}
}

package main

import (
	"testing"
	"time"

	"github.com/beilunyang/openhongguo/sdk/model"
	"github.com/egoist/mygo/ui"
)

// sidebarApp is a test app whose sections all have what they show, so
// visiting one asks the network for nothing.
func sidebarApp(t *testing.T) (*app, *ui.Tester) {
	a := testApp(t)
	for _, key := range []string{"rank::", "new:", "search:归聿"} {
		f := newFeed(nil)
		f.started = true
		a.feeds[key] = f
	}
	a.hot = &model.HotSearch{}
	tt := ui.NewTester(a.view, 1180, 760)
	tt.Advance(time.Second)
	return a, tt
}

func click(t *testing.T, tt *ui.Tester, s string) {
	t.Helper()
	if err := tt.Click(s); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
}

// A series opened from 推荐 leaves 推荐 chosen in the sidebar, and a click
// on it goes back there.
func TestSidebarReturnsFromSeries(t *testing.T) {
	a, tt := sidebarApp(t)
	click(t, tt, "浅浅归聿")
	if got := a.router.Location(); got != "/series/1" {
		t.Fatalf("at %q after a click on a series", got)
	}
	click(t, tt, "推荐")
	if got := a.router.Location(); got != "/discover" {
		t.Fatalf("at %q after a click on 推荐", got)
	}
}

// Search results show under 热搜, and a click on it goes to the hot
// searches.
func TestSidebarReturnsFromSearch(t *testing.T) {
	a, tt := sidebarApp(t)
	a.router.Push("/search?q=归聿")
	tt.Advance(time.Second)
	click(t, tt, "热搜")
	if got := a.router.Location(); got != "/search" {
		t.Fatalf("at %q after a click on 热搜", got)
	}
}

// A click on another section goes there once: Back returns to where the
// viewer was, and a click on the page shown already adds nothing.
func TestSidebarPushesOnce(t *testing.T) {
	a, tt := sidebarApp(t)
	click(t, tt, "排行榜")
	click(t, tt, "排行榜")
	if got := a.router.Location(); got != "/rank" {
		t.Fatalf("at %q after a click on 排行榜", got)
	}
	a.router.Back()
	tt.Advance(time.Second)
	if got := a.router.Location(); got != "/discover" {
		t.Fatalf("Back went to %q", got)
	}
}

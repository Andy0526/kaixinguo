package main

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// playerApp is the test app playing its first episode, without a web view
// or a window.
func playerApp(t *testing.T) (*app, *ui.Tester) {
	a := testApp(t)
	a.cur = &playback{seriesID: "1", number: 1, gen: 1}
	a.router.Push("/play/1")
	tt := ui.NewTester(a.view, 1180, 760)
	tt.SetTitleBar(ui.TitleBar{Height: titleBarHeight, Left: 80})
	tt.Advance(time.Second)
	return a, tt
}

// The theater fills the screen: the episodes leave the side and come back
// from the right edge, out only when the pointer nears it.
func TestPlayerTheater(t *testing.T) {
	a, tt := playerApp(t)
	for _, s := range []string{"退出播放", "影院", "第 1 集 / 共 130 集"} {
		if !tt.HasText(s) {
			t.Fatalf("no %q in %q", s, tt.Texts())
		}
	}
	click(t, tt, "影院")
	if !a.fullscreen || !a.theaterShown || a.panelOpen {
		t.Fatalf("theater: full screen %v, shown %v, panel open %v", a.fullscreen, a.theaterShown, a.panelOpen)
	}
	if !tt.HasText("退出影院") {
		t.Fatalf("no theater button in %q", tt.Texts())
	}

	// The pointer near the right edge brings the episodes out, and
	// leaving them puts them away.
	tt.Move(1170, 380)
	tt.Advance(time.Second)
	if !a.panelOpen || !tt.HasText("第 1 集 / 共 130 集") {
		t.Fatalf("the episodes do not show: panel %v, texts %q", a.panelOpen, tt.Texts())
	}
	tt.Move(600, 380)
	tt.Advance(time.Second)
	if a.panelOpen {
		t.Fatal("the episodes stay after the pointer left them")
	}

	tt.Move(1170, 380)
	tt.Advance(time.Second)
	click(t, tt, "收起选集")
	if a.panelOpen || tt.HasText("收起选集") {
		t.Fatal("the episodes stay after the close button")
	}

	// A close by hand holds even with the pointer still at the edge, and
	// leaving clears that hold.
	tt.Move(600, 380)
	tt.Advance(time.Second)
	tt.Move(1170, 380)
	tt.Advance(time.Second)
	if !a.panelOpen {
		t.Fatal("the episodes do not come back after the pointer left")
	}
	tt.ClickAt(20, 300) // the dimmed backdrop
	tt.Advance(time.Second)
	if a.panelOpen {
		t.Fatal("the episodes stay after a click outside them")
	}

	click(t, tt, "退出影院")
	if a.fullscreen || a.theaterShown || a.panelOpen {
		t.Fatalf("out of theater: full screen %v, shown %v, panel open %v", a.fullscreen, a.theaterShown, a.panelOpen)
	}
	if !tt.HasText("影院") || !tt.HasText("第 1 集 / 共 130 集") {
		t.Fatalf("the side panel did not come back: %q", tt.Texts())
	}

	click(t, tt, "退出播放")
	if got := a.router.Path(); got != "/discover" {
		t.Fatalf("at %q after leaving the player", got)
	}
}

// Escape closes the episodes first, then leaves the theater, and only then
// the player.
func TestPlayerEscape(t *testing.T) {
	a, tt := playerApp(t)
	click(t, tt, "影院")
	tt.Move(1170, 380)
	tt.Advance(time.Second)
	if !a.panelOpen {
		t.Fatal("the pointer at the right edge did not show the episodes")
	}

	tt.Key(0, ui.KeyEscape)
	tt.Advance(time.Second)
	if a.panelOpen {
		t.Fatal("Escape did not close the episodes")
	}
	if !a.fullscreen {
		t.Fatal("Escape left the theater with the episodes")
	}
	if got := a.router.Path(); got != "/play/1" {
		t.Fatalf("at %q after closing the episodes", got)
	}

	tt.Key(0, ui.KeyEscape)
	tt.Advance(time.Second)
	if a.fullscreen {
		t.Fatal("Escape did not leave the theater")
	}
	if got := a.router.Path(); got != "/play/1" {
		t.Fatalf("at %q after leaving the theater", got)
	}

	tt.Key(0, ui.KeyEscape)
	tt.Advance(time.Second)
	if got := a.router.Path(); got != "/discover" {
		t.Fatalf("at %q after leaving the player", got)
	}
}

// The page's double click asks for the theater; the window follows.
func TestPlayerDoubleClickTheater(t *testing.T) {
	a := testApp(t)
	a.cur = &playback{seriesID: "1", number: 1, gen: 1}
	a.router.Push("/play/1")
	a.onPlayerEvent(url.Values{"type": {"theater"}, "ep": {"e1"}})
	if !a.fullscreen {
		t.Fatal("the double click did not enter the theater")
	}
	a.onPlayerEvent(url.Values{"type": {"theater"}, "ep": {"e1"}})
	if a.fullscreen {
		t.Fatal("the double click did not leave the theater")
	}
}

// The player's controls are the app's bottom bar, not the video's native
// ones, which put the small window and full screen buttons over the picture.
func TestPlayerControlsAreTheBottomBar(t *testing.T) {
	for _, want := range []string{`id="bar"`, `id="seek"`, `id="vol"`, `id="pip"`, `<video id="v" playsinline`} {
		if !strings.Contains(playerHTML, want) {
			t.Fatalf("the player page has no %s", want)
		}
	}
	if strings.Contains(playerHTML, `<video id="v" controls`) {
		t.Fatal("the player page still uses the native video controls")
	}
}

// The player's exit and the series' back stay clear of the traffic lights.
func TestTitleBarButtonsClearOfTrafficLights(t *testing.T) {
	a := testApp(t)
	a.router.Push("/series/1")
	tt := ui.NewTester(a.view, 1180, 760)
	tt.SetTitleBar(ui.TitleBar{Height: titleBarHeight, Left: 80})
	tt.Advance(time.Second)
	if r, ok := tt.Find("返回"); !ok || r.X < 80 {
		t.Fatalf("the series' back button at %v, under the traffic lights", r)
	}

	a.cur = &playback{seriesID: "1", number: 1, gen: 1}
	a.router.Push("/play/1")
	tt.Advance(time.Second)
	if r, ok := tt.Find("退出播放"); !ok || r.X < 80 {
		t.Fatalf("the player's exit button at %v, under the traffic lights", r)
	}
	tt.SetTitleBar(ui.TitleBar{Height: titleBarHeight})
	tt.Advance(time.Second)
	if r, ok := tt.Find("退出播放"); !ok || r.X != 16 {
		t.Fatalf("the player's exit button at %v in full screen, not at 16", r)
	}
}

// The theater's episodes come from the right edge, over the video.
func TestTheaterDrawerOnTheRight(t *testing.T) {
	_, tt := playerApp(t)
	click(t, tt, "影院")
	tt.Move(1170, 380)
	tt.Advance(time.Second)
	r, ok := tt.Find("第 1 集 / 共 130 集")
	if !ok || r.X < 880 {
		t.Fatalf("the episodes at %v, not on the right", r)
	}
	if c, ok := tt.Find("收起选集"); !ok || c.X < 1100 {
		t.Fatalf("the close button at %v, not in the drawer", c)
	}
}

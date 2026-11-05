// 开心果 is a small, fast desktop player for Hongguo short dramas, built on
// the openhongguo SDK and MyGo's native UI.
package main

import (
	"log"
	"net/url"
	"path/filepath"

	"github.com/beilunyang/openhongguo/sdk"
	"github.com/beilunyang/openhongguo/sdk/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	titleBarHeight = 52
	sidebarWidth   = 220
)

// app is the state the window shows. Everything in it belongs to the main
// thread: work on other goroutines hands its results over with update.
type app struct {
	win    *mygo.Window
	ready  chan struct{} // closed once win is set
	router *ui.Router
	client *sdk.Client
	covers *covers
	hist   *history
	proxy  *proxy

	cacheDir string

	// The sidebar.
	query       string
	searchField ui.Handle
	section     string // the sidebar's page the page shown belongs to
	recentOpen  bool
	sideShown   float32

	// The catalog.
	home         *feed
	feeds        map[string]*feed
	rankContent  string
	rankBoard    string
	rankIdx      int
	rankContents []model.RankContent
	newDate      string
	newDates     []string
	hot          *model.HotSearch
	hotLoading   bool
	hotErr       error

	// Series and playback.
	series  map[string]*seriesPage
	futures map[string]*future
	cur     *playback
	gen     int

	// The player's web view.
	web       *mygo.WebView
	webReady  bool
	pendingJS string
	js        chan string // what the page runs, in order
	onPlayer  bool
}

func newApp(dataDir, cacheDir string) *app {
	a := &app{
		ready:      make(chan struct{}),
		router:     ui.NewRouter("/discover"),
		cacheDir:   cacheDir,
		section:    "/discover",
		recentOpen: true,
		sideShown:  1,
		feeds:      map[string]*feed{},
		series:     map[string]*seriesPage{},
		futures:    map[string]*future{},
		hist:       loadHistory(dataDir),
	}
	a.covers = newCovers(filepath.Join(cacheDir, "covers"), a.update)
	a.home = newFeed(a.discover)
	// Last launch's recommendations show at once; fresh ones replace them.
	if cards := a.loadHomeCache(); len(cards) > 0 {
		a.home.add(page{cards: cards, more: true})
		a.home.stale = true
	}
	a.home.onPage = func(page) { a.saveHomeCache(a.home.cards) }
	return a
}

// update runs fn on the main thread and redraws, once the window exists.
func (a *app) update(fn func()) {
	<-a.ready
	a.win.Update(fn)
}

func main() {
	dataDir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		log.Fatal(err)
	}
	cacheDir, err := mygo.App.Path(mygo.PathCache)
	if err != nil {
		log.Fatal(err)
	}
	a := newApp(dataDir, cacheDir)
	if a.client, err = newClient(dataDir); err != nil {
		log.Fatal(err)
	}
	if a.proxy, err = newProxy(func(q url.Values) { a.update(func() { a.onPlayerEvent(q) }) }); err != nil {
		log.Fatal(err)
	}
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:          "开心果",
			Width:          1180,
			Height:         760,
			MinWidth:       780,
			MinHeight:      520,
			StateKey:       "main",
			TitleBarStyle:  mygo.TitleBarHiddenInset,
			TitleBarHeight: titleBarHeight,
			Vibrancy:       mygo.VibrancySidebar,
			Content:        ui.View(a.view),
		})
		close(a.ready)
		a.loadMore(a.home)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

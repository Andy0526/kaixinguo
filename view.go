package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/beilunyang/openhongguo/sdk/model"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

const (
	gridPadX   = 22  // the grid's margin, outside its cells' own
	cellPadX   = 8   // room on each side of a poster in its cell
	posterMinW = 150 // the narrowest a cell gets
)

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	vibrant := c.Vibrancy()
	if vibrant {
		c.Root().Background(ui.Transparent)
	}
	path := a.router.Path()
	onPlayer := strings.HasPrefix(path, "/play/")
	if a.onPlayer && !onPlayer {
		a.evalJS("kx.pause()") // left the player
	}
	a.onPlayer = onPlayer
	switch path {
	case "/discover", "/rank", "/new", "/search":
		a.section = path
	}

	row := ui.Row(c).Fill().AlignItems(ui.Stretch)
	a.sideShown = row.Animate("sidebar", b2f(!onPlayer), 280*time.Millisecond)
	row.Children(func() {
		// The sidebar slides away while the player shows.
		w := sidebarWidth * a.sideShown
		ui.Row(c).Width(w).Shrink(0).AlignItems(ui.Stretch).Clip().Children(func() {
			side := ui.Column(c).Width(sidebarWidth).Shrink(0).Margin(0, 0, 0, w-sidebarWidth)
			if !vibrant {
				side.Background(t.Surface)
			}
			side.Children(func() { a.sidebar(c, vibrant) })
		})
		if w > 0.5 {
			ui.Box(c).Width(1).Background(hairline(t))
		}
		a.router.View(c, func(r *ui.Route) {
			switch {
			case r.Match("/discover"):
				r.Title("推荐")
				a.gridPage(c, "推荐", a.home, nil, nil, 0, "暂时没有推荐")
			case r.Match("/rank"):
				r.Title("排行榜")
				a.rankPage(c)
			case r.Match("/new"):
				r.Title("新剧上线")
				a.newPage(c)
			case r.Match("/search"):
				r.Title("搜索")
				a.searchPage(c, r.Query("q"))
			case r.Match("/series/{id}"):
				r.Title("详情")
				a.seriesView(c, r.Param("id"))
			case r.Match("/play/{id}"):
				r.Title("播放")
				a.playerView(c, r.Param("id"))
			}
		}).Grow(1).MinWidth(0).Background(t.Background)
	})

	c.OnShortcut(ui.Cmd, ui.KeyF, func() {
		if onPlayer {
			a.router.Back()
		}
		a.searchField.Focus(c)
	})
}

func (a *app) sidebar(c *ui.Context, vibrant bool) {
	t := c.Theme()
	ui.Box(c).Height(titleBarHeight).Shrink(0).DragWindow()
	field := ui.SearchField(c.Key("search"), &a.query).Placeholder("搜索短剧").Label("搜索短剧").
		Bind(&a.searchField).Margin(0, 12, 10, 12)
	if field.Submitted() {
		a.router.Push("/search?q=" + url.QueryEscape(strings.TrimSpace(a.query)))
	}

	selected := a.section
	path := a.router.Path()
	if id, ok := strings.CutPrefix(path, "/series/"); ok {
		if _, watched := a.hist.get(id); watched {
			selected = path
		}
	}
	list := ui.Sidebar(c, &selected, func() {
		ui.SidebarSection(c, "发现", nil, func() {
			ui.SidebarItem(c, "/discover", iconHome, "推荐")
			ui.SidebarItem(c, "/rank", iconChart, "排行榜")
			ui.SidebarItem(c, "/new", iconCalendar, "新剧上线")
			ui.SidebarItem(c, "/search", iconFlame, "热搜")
		})
		if len(a.hist.list) > 0 {
			ui.SidebarSection(c, "最近观看", &a.recentOpen, func() {
				for _, w := range a.hist.list[:min(len(a.hist.list), 8)] {
					ui.SidebarItem(c.Key(w.SeriesID), "/series/"+w.SeriesID, iconClock, w.Title).Children(func() {
						ui.Textf(c, "%d", w.Episode).FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
					})
				}
			})
		}
	}).Grow(1).Label("导航")
	if vibrant {
		list.Background(ui.Transparent)
	}
	if list.Changed() {
		a.router.Push(selected)
	}
}

// header is a page's title, level with the window controls, and what goes
// beside it.
func (a *app) header(c *ui.Context, title string, right func()) {
	ui.Row(c).Height(titleBarHeight).Shrink(0).AlignItems(ui.Center).Gap(12).
		Padding(0, gridPadX+cellPadX, 0, gridPadX+cellPadX).DragWindow().Children(func() {
		ui.Text(c, title).FontSize(22).Bold().SingleLine()
		ui.Spacer(c)
		if right != nil {
			right()
		}
	})
}

// gridPage is a page of posters under a header that the grid scrolls
// under, with a row of controls under the title when it has them.
func (a *app) gridPage(c *ui.Context, title string, f *feed, right, controls func(), controlsH float32, empty string) {
	top := float32(titleBarHeight) + controlsH
	ui.Box(c).Fill().Children(func() {
		a.cardGrid(c, f, top+4, empty)
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(top + 12).PassThrough().Material(glass.ScrollEdge{})
		ui.Column(c).Absolute().Top(0).Left(0).Right(0).Children(func() {
			a.header(c, title, right)
			if controls != nil {
				ui.Row(c).Height(controlsH).AlignItems(ui.Start).Padding(0, gridPadX+cellPadX-2).Children(controls)
			}
		})
	})
}

func (a *app) cardGrid(c *ui.Context, f *feed, top float32, empty string) {
	if len(f.cards) == 0 {
		a.feedState(c, f, top, empty)
		return
	}
	// The rows' height: as GridView lays the cells out, a poster of 3:4 in
	// each, and its two lines.
	w, _ := c.Size()
	inner := w - sidebarWidth*a.sideShown - 1 - 2*gridPadX
	gap := c.Theme().Space(2)
	cols := max(1, int((inner+gap)/(posterMinW+gap)))
	itemW := (inner - float32(cols-1)*gap) / float32(cols)
	rowH := (itemW-2*cellPadX)*4/3 + 62
	n := len(f.cards)
	ui.GridView(c, &f.grid, n, posterMinW, rowH, func(i int) {
		a.card(c, f.cards[i])
		if i >= n-cols*2 && f.more {
			a.loadMore(f)
		}
	}).Fill().Padding(top, gridPadX, 24, gridPadX).ScrollbarInsets(top, 0, 0, 0)
}

// feedState is what a feed with no series yet shows.
func (a *app) feedState(c *ui.Context, f *feed, top float32, empty string) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(top, 24, 24, 24).Center().Gap(10).Children(func() {
		switch {
		case f.err != nil && f.tries < 4:
			// Try again by itself, after 2, 4, 8 and 16 seconds.
			wait := time.Duration(2<<f.tries) * time.Second
			if since := time.Since(f.errAt); since >= wait {
				f.tries++
				a.retry(f)
			} else {
				c.After(wait - since)
			}
			ui.Spinner(c).FontSize(20).Label("正在重试")
			ui.Text(c, "网络不稳定，正在重试…").FontSize(13).TextColor(t.TextMuted)
		case f.err != nil:
			ui.Text(c, "加载失败").FontSize(15).FontWeight(600)
			ui.Text(c, friendly(f.err)).FontSize(13).TextColor(t.TextMuted).TextAlign(ui.Center).MaxWidth(360)
			ui.Button(c.Key("retry"), "重试").Margin(6, 0, 0, 0).OnClick(func() { f.tries = 0; a.retry(f) })
		case f.loading || !f.started:
			ui.Spinner(c).FontSize(20).Label("正在加载")
			if !f.started {
				a.loadMore(f)
			}
		default:
			ui.Text(c, empty).FontSize(14).TextColor(t.TextMuted)
		}
	})
}

// card is a series in a grid: its poster, title and a line about it.
func (a *app) card(c *ui.Context, k card) {
	t := c.Theme()
	b := ui.ButtonBase(c.Key(k.ID)).Fill().Label(k.Title).Cursor(ui.CursorPointer)
	hover := b.Hovered()
	b.Children(func() {
		ui.Column(c).Fill().Padding(4, cellPadX, 14, cellPadX).Gap(9).Children(func() {
			a.poster(c, k.Cover, 10, hover).Grow(1).MinHeight(0)
			ui.Column(c).Gap(2).Children(func() {
				ui.Row(c).Gap(6).Children(func() {
					if k.Rank > 0 {
						ui.Textf(c, "%d", k.Rank).FontSize(13).Bold().TextColor(t.TextMuted).FontFeatures("tnum")
					}
					ui.Text(c, k.Title).FontSize(13).FontWeight(600).SingleLine().Grow(1).MinWidth(0)
				})
				ui.Text(c, k.Sub).FontSize(12).TextColor(t.TextMuted).SingleLine()
			})
		})
	})
	if b.Clicked() {
		a.router.Push("/series/" + k.ID)
	}
}

// poster is a cover in a rounded frame, gray until it loads, lifting a
// little under the pointer.
func (a *app) poster(c *ui.Context, cover string, radius float32, hover bool) ui.Element {
	t := c.Theme()
	p := ui.Box(c).Radius(radius).Clip().Background(placeholder(t))
	lift := p.Animate("lift", b2f(hover), 180*time.Millisecond)
	s := 1 + 0.025*lift
	p.Scale(s, s).Shadow(0, 2+5*lift, 8+14*lift, 0, ui.RGBA(0, 0, 0, 0.12+0.12*lift))
	p.Children(func() {
		if bmp, alpha := a.covers.get(c, cover); bmp != nil {
			ui.Image(c, bmp).Fill().Fit(ui.Cover).Opacity(alpha)
		}
	})
	return p
}

// feed returns the feed of key, making it with mk the first time.
func (a *app) feed(key string, mk func() *feed) *feed {
	f := a.feeds[key]
	if f == nil {
		f = mk()
		a.feeds[key] = f
	}
	return f
}

func (a *app) rankPage(c *ui.Context) {
	t := c.Theme()
	content, board := a.rankContent, a.rankBoard
	f := a.feed("rank:"+content+":"+board, func() *feed {
		f := newFeed(a.rankFetcher(content, board))
		f.onPage = func(p page) {
			if len(p.contents) == 0 {
				return
			}
			a.rankContents = p.contents
			if a.rankContent == "" { // the segment of the server's choice
				n := 0
				for _, rc := range p.contents {
					if rc.ID == "" || len(rc.Boards) == 0 || strings.Contains(rc.Name, "演员") {
						continue
					}
					if rc.Selected {
						a.rankIdx = n
					}
					n++
				}
			}
		}
		return f
	})
	// Contents with charts of series; the actors' chart has none.
	var contents []model.RankContent
	for _, rc := range a.rankContents {
		if rc.ID != "" && len(rc.Boards) > 0 && !strings.Contains(rc.Name, "演员") {
			contents = append(contents, rc)
		}
	}
	right := func() {
		if len(contents) < 2 {
			return
		}
		labels := make([]string, len(contents))
		for i, rc := range contents {
			labels[i] = rc.Name
		}
		a.rankIdx = min(a.rankIdx, len(contents)-1)
		ui.Segmented(c.Key("contents"), &a.rankIdx, labels...)
		if id := contents[a.rankIdx].ID; id != a.currentRankContent(contents) {
			a.rankContent, a.rankBoard = id, ""
		}
	}
	controls := func() {
		var boards []model.RankBoard
		cur := a.currentRankContent(contents)
		for _, rc := range contents {
			if rc.ID == cur {
				boards = rc.Boards
			}
		}
		ui.ScrollHorizontal(c).Grow(1).Scrollbars(false).Children(func() {
			ui.Row(c).Gap(4).Children(func() {
				for _, b := range boards {
					on := b.ID == a.rankBoard || (a.rankBoard == "" && b.Selected)
					if a.chip(c, "board:"+b.ID, b.Name, on, t) {
						a.rankBoard = b.ID
					}
				}
			})
		})
	}
	a.gridPage(c, "排行榜", f, right, controls, 40, "这个榜单暂时是空的")
}

// currentRankContent is the content whose charts show: the one chosen,
// else the one the server chose.
func (a *app) currentRankContent(contents []model.RankContent) string {
	if a.rankContent != "" {
		return a.rankContent
	}
	for _, rc := range contents {
		if rc.Selected {
			return rc.ID
		}
	}
	if len(contents) > 0 {
		return contents[0].ID
	}
	return ""
}

// chip is a capsule that chooses one of a few, as a chart or a day.
func (a *app) chip(c *ui.Context, key, label string, on bool, t *ui.Theme) bool {
	b := ui.ButtonBase(c.Key(key)).Padding(5, 12).Radius(999).Label(label).Cursor(ui.CursorPointer)
	switch {
	case on:
		b.Background(t.Text).TextColor(t.Background)
	case b.Hovered():
		b.Background(t.Text.Alpha(0.08))
	}
	b.Children(func() { ui.Text(c, label).FontSize(13).FontWeight(500).SingleLine() })
	return b.Clicked() && !on
}

func (a *app) newPage(c *ui.Context) {
	t := c.Theme()
	date := a.newDate
	f := a.feed("new:"+date, func() *feed {
		f := newFeed(a.newFetcher(date))
		f.onPage = func(p page) {
			if len(p.dates) > 0 {
				a.newDates = p.dates
			}
		}
		return f
	})
	today := time.Now().Format("20060102")
	controls := func() {
		ui.ScrollHorizontal(c).Grow(1).Scrollbars(false).Children(func() {
			ui.Row(c).Gap(4).Children(func() {
				for _, d := range a.newDates {
					on := d == a.newDate || (a.newDate == "" && d == today)
					if a.chip(c, "day:"+d, dayLabel(d, today), on, t) {
						a.newDate = d
						if d == today {
							a.newDate = ""
						}
					}
				}
			})
		})
	}
	a.gridPage(c, "新剧上线", f, nil, controls, 40, "这一天没有新剧")
}

// dayLabel names a day of the calendar (YYYYMMDD) as people say it.
func dayLabel(d, today string) string {
	day, err := time.ParseInLocation("20060102", d, time.Local)
	now, err2 := time.ParseInLocation("20060102", today, time.Local)
	if err != nil || err2 != nil {
		return d
	}
	switch int(day.Sub(now).Hours() / 24) {
	case 0:
		return "今天"
	case 1:
		return "明天"
	case -1:
		return "昨天"
	}
	return fmt.Sprintf("%d月%d日", day.Month(), day.Day())
}

func (a *app) searchPage(c *ui.Context, q string) {
	q = strings.TrimSpace(q)
	if q == "" {
		a.hotPage(c)
		return
	}
	f := a.feed("search:"+q, func() *feed {
		f := newFeed(a.searchFetcher(q))
		f.order = orderSeasons
		return f
	})
	a.gridPage(c, "“"+q+"”", f, nil, nil, 0, "没有找到“"+q+"”相关的短剧，换个词试试")
}

// hotPage shows what people search for, before a search.
func (a *app) hotPage(c *ui.Context) {
	t := c.Theme()
	if a.hot == nil && !a.hotLoading && a.hotErr == nil {
		a.loadHot()
	}
	ui.Box(c).Fill().Children(func() {
		top := float32(titleBarHeight)
		switch {
		case a.hot == nil && a.hotErr != nil:
			ui.Column(c).Fill().Center().Gap(10).Children(func() {
				ui.Text(c, "加载失败").FontSize(15).FontWeight(600)
				ui.Text(c, friendly(a.hotErr)).FontSize(13).TextColor(t.TextMuted)
				ui.Button(c.Key("retry"), "重试").OnClick(func() { a.hotErr = nil; a.loadHot() })
			})
		case a.hot == nil:
			ui.Column(c).Fill().Center().Children(func() { ui.Spinner(c).FontSize(20).Label("正在加载") })
		default:
			ui.Scroll(c).Fill().Padding(top, gridPadX, 28, gridPadX).ScrollbarInsets(top, 0, 0, 0).Children(func() {
				for _, b := range a.hot.Boards {
					if len(b.Items) == 0 {
						continue
					}
					ui.Text(c, b.Name).FontSize(17).Bold().Margin(14, cellPadX, 10, cellPadX)
					ui.Row(c).Wrap().Children(func() {
						for _, it := range b.Items {
							k := card{ID: it.SeriesID, Title: it.Title, Cover: it.CoverURL, Sub: it.SearchScore, Rank: it.Position}
							if k.Sub == "" {
								k.Sub = it.Heat
							}
							ui.Box(c.Key(b.Name+it.SeriesID)).Size(166, (166-2*cellPadX)*4/3+62).Children(func() { a.card(c, k) })
						}
					})
				}
			})
		}
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(top + 12).PassThrough().Material(glass.ScrollEdge{})
		ui.Column(c).Absolute().Top(0).Left(0).Right(0).Children(func() { a.header(c, "热搜", nil) })
	})
}

func (a *app) loadHot() {
	a.hotLoading = true
	go func() {
		ctx, cancel := contextTimeout(60 * time.Second)
		defer cancel()
		h, err := a.client.HotSearch(ctx)
		a.update(func() { a.hot, a.hotErr, a.hotLoading = h, err, false })
	}()
}

func hairline(t *ui.Theme) ui.Color    { return t.Text.Alpha(0.1) }
func placeholder(t *ui.Theme) ui.Color { return t.Text.Alpha(0.07) }

func b2f(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

func itoa(n int) string { return strconv.Itoa(n) }

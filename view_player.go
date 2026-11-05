package main

import (
	"github.com/beilunyang/openhongguo/sdk/model"
	"github.com/egoist/mygo/ui"
)

var black = ui.RGB(0, 0, 0)

// playerView is the episode playing, full height, beside the episodes.
func (a *app) playerView(c *ui.Context, id string) {
	sp := a.seriesPage(id)
	if a.cur == nil || a.cur.seriesID != id {
		// Reached through history: go on where the viewer was.
		if sp.series != nil && len(sp.episodes) > 0 {
			sid, n, start, _ := a.resumePoint(sp)
			a.play(sid, n, start)
		}
	}
	dark := *ui.DarkTheme()
	dark.Accent, dark.AccentText = c.Theme().Accent, c.Theme().AccentText
	ui.Themed(c, &dark).Fill().Background(black).Children(func() {
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Background(black).Children(func() {
				if a.web != nil {
					ui.WebView(c, a.web).Fill()
				}
				a.playerOverlay(c)
				ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(titleBarHeight).DragWindow()
				a.backButton(c, c.TitleBar().Left+10)
			})
			a.playerPanel(c, sp)
		})
	})
	c.OnShortcut(0, ui.KeySpace, func() { a.evalJS("kx.toggle()") })
	c.OnShortcut(0, ui.KeyLeft, func() { a.evalJS("kx.seek(-5)") })
	c.OnShortcut(0, ui.KeyRight, func() { a.evalJS("kx.seek(5)") })
	c.OnShortcut(0, ui.KeyDown, func() { a.step(1) })
	c.OnShortcut(0, ui.KeyUp, func() { a.step(-1) })
	c.OnShortcut(0, ui.KeyEscape, func() { a.router.Back() })
}

// playerOverlay is what covers the video while it isn't playing: a
// spinner while it loads, or what went wrong.
func (a *app) playerOverlay(c *ui.Context) {
	p := a.cur
	if p == nil || p.state == playPlaying {
		return
	}
	t := c.Theme()
	ov := ui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Center().Gap(14)
	if p.state == playFailed {
		ov.Background(ui.RGBA(0, 0, 0, 0.82))
	} else {
		ov.PassThrough()
		if p.state == playResolving {
			ov.Background(ui.RGBA(0, 0, 0, 0.5))
		}
	}
	ov.Children(func() {
		if p.state != playFailed {
			ui.Spinner(c).FontSize(26).TextColor(ui.RGBA(255, 255, 255, 0.85)).Label("正在加载")
			return
		}
		ui.Text(c, p.msg).FontSize(15).TextColor(t.Text).TextAlign(ui.Center).MaxWidth(420)
		ui.Row(c).Gap(10).Children(func() {
			if ui.Button(c.Key("retry"), "重试").Clicked() {
				a.play(p.seriesID, p.number, p.start)
			}
			if sp := a.series[p.seriesID]; sp != nil {
				if _, ok := sp.episode(p.number + 1); ok {
					if ui.Button(c.Key("next"), "下一集").Clicked() {
						a.step(1)
					}
				}
			}
		})
	})
}

// playerPanel is the series' title and its episodes, beside the video.
func (a *app) playerPanel(c *ui.Context, sp *seriesPage) {
	t := c.Theme()
	ui.Column(c).Width(296).Shrink(0).Background(ui.RGB(22, 22, 24)).Children(func() {
		ui.Box(c).Height(titleBarHeight).Shrink(0).DragWindow()
		if sp.series == nil {
			ui.Column(c).Grow(1).Center().Children(func() { ui.Spinner(c).FontSize(18) })
			return
		}
		cur := 0
		if a.cur != nil && a.cur.seriesID == sp.id {
			cur = a.cur.number
		}
		ui.Column(c).Padding(0, 22, 16, 22).Gap(6).Children(func() {
			ui.Text(c, sp.series.Title).FontSize(17).Bold().MaxLines(2)
			ui.Textf(c, "第 %d 集 / 共 %d 集", cur, len(sp.episodes)).FontSize(12).TextColor(t.TextMuted).FontFeatures("tnum")
			if len(sp.seasons) > 1 {
				ui.Box(c).Height(4)
				a.seasonPicker(c, sp, func(se model.Season) { a.playSeason(se.SeriesID) })
			}
			a.partPicker(c, sp)
		})
		ui.Divider(c).Background(ui.RGBA(255, 255, 255, 0.06))
		// Bring the episode playing into view as it changes.
		reveal := cur > 0 && cur != sp.lastShown
		if reveal {
			sp.part = (cur - 1) / episodesPerPart
			sp.lastShown = cur
		}
		lo, hi := a.partRange(sp)
		ui.Scroll(c).Grow(1).Padding(14, 16, 22, 16).Children(func() {
			ui.Row(c).Wrap().Gap(8).Children(func() {
				for _, e := range sp.episodes[lo:hi] {
					chip := a.episodeChip(c, e, e.Number == cur, t).Size(44, 38)
					if reveal && e.Number == cur {
						chip.ScrollIntoView()
					}
					if chip.Clicked() && e.Playable && e.Number != cur {
						a.play(sp.id, e.Number, 0)
					}
				}
			})
		})
		ui.Text(c, "空格 暂停　← → 快进快退　↑ ↓ 换集").FontSize(11).TextColor(t.TextMuted.Alpha(0.8)).
			Padding(10, 22, 16, 22).SingleLine()
	})
}

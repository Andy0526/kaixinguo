package main

import (
	"fmt"
	"strings"

	"github.com/beilunyang/openhongguo/sdk/model"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

const episodesPerPart = 100

func (a *app) seriesView(c *ui.Context, id string) {
	t := c.Theme()
	sp := a.seriesPage(id)
	ui.Box(c).Fill().Children(func() {
		switch {
		case sp.series == nil && sp.err != nil:
			ui.Column(c).Fill().Center().Gap(10).Children(func() {
				ui.Text(c, "加载失败").FontSize(15).FontWeight(600)
				ui.Text(c, friendly(sp.err)).FontSize(13).TextColor(t.TextMuted)
				ui.Button(c.Key("retry"), "重试").OnClick(func() { a.loadSeries(sp) })
			})
		case sp.series == nil:
			ui.Column(c).Fill().Center().Children(func() { ui.Spinner(c).FontSize(20).Label("正在加载") })
		default:
			a.seriesBody(c, sp)
		}
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(titleBarHeight).DragWindow()
		a.backButton(c)
	})
}

// backButton is a round button of glass that goes back, clear of the
// window's traffic lights.
func (a *app) backButton(c *ui.Context) {
	if !a.router.CanGoBack() {
		return
	}
	b := ui.ButtonBase(c.Key("back")).Absolute().Top((titleBarHeight-30)/2).Left(titleBarLeft(c, 10)).Size(30, 30).Radius(15).
		Center().Material(glass.Glass{Interactive: true}).Label("返回").Tooltip("返回").Cursor(ui.CursorPointer)
	b.Children(func() { ui.Icon(c, iconBack).Size(15, 15) })
	if b.Clicked() {
		a.router.Back()
	}
}

func (a *app) seriesBody(c *ui.Context, sp *seriesPage) {
	t := c.Theme()
	s := sp.series
	bmp, alpha := a.covers.get(c, s.CoverURL)
	ui.Scroll(c).Fill().Children(func() {
		// The hero: the poster over its own colors, blurred wide. Its
		// layers are all positioned, so they paint in the order built.
		const posterW, posterH = 200, 267
		heroH := float32(titleBarHeight + 24 + posterH + 30)
		ui.Box(c).Height(heroH).Shrink(0).Children(func() {
			// Only the blur is cut to the hero: the poster's shadow falls
			// below it.
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Clip().Children(func() {
				if bmp != nil {
					ui.Image(c, bmp).Absolute().Top(-90).Left(-90).Right(-90).Bottom(-90).Fit(ui.Cover).
						Blur(56).Opacity(alpha).Cache()
				}
			})
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).LinearGradient(ui.LinearGradient{
				From: t.Background.Alpha(0.15), To: t.Background, Angle: 180, Start: 0.1, End: 0.97, Oklab: true,
			})
			ui.Row(c).Absolute().Left(0).Right(0).Bottom(0).Padding(0, 44, 30, 44).Gap(32).AlignItems(ui.End).Children(func() {
				ui.Box(c).Size(posterW, posterH).Shrink(0).Radius(12).Clip().Background(placeholder(t)).
					Shadow(0, 16, 36, 0, ui.RGBA(0, 0, 0, 0.32)).Children(func() {
					if bmp != nil {
						ui.Image(c, bmp).Fill().Fit(ui.Cover).Opacity(alpha)
					}
				})
				ui.Column(c).Grow(1).MinWidth(0).Gap(10).Children(func() {
					ui.Text(c, s.Title).FontSize(30).Bold().MaxLines(2).LetterSpacing(-0.3)
					ui.Text(c, metaLine(s)).FontSize(13).TextColor(t.TextMuted).SingleLine()
					if s.Description != "" {
						ui.Text(c, s.Description).FontSize(13).LineHeight(1.6).MaxLines(4).MaxWidth(640).
							TextColor(t.Text.Alpha(0.86)).Selectable()
					}
					ui.Row(c).Gap(10).Margin(10, 0, 0, 0).Children(func() {
						if len(sp.episodes) > 0 {
							id, n, start, label := a.resumePoint(sp)
							if a.playButton(c, label) {
								a.play(id, n, start)
							}
						}
						a.favButton(c, sp)
					})
				})
			})
		})
		a.episodeGrid(c, sp)
	})
}

// metaLine is the series at a glance: its kind, its episodes, its heat.
func metaLine(s *model.Series) string {
	var parts []string
	for _, r := range s.Rows {
		if r.Kind == model.InfoCategory && r.Text != "" {
			parts = append(parts, r.Text)
			break
		}
	}
	switch s.ContentType {
	case model.ContentMotionComic:
		parts = append(parts, "漫剧")
	case model.ContentMovie:
		parts = append(parts, "电影")
	}
	if s.EpisodeCount > 0 {
		parts = append(parts, episodesLabel(s))
	}
	if s.Heat > 0 {
		parts = append(parts, "热度 "+heat(s.Heat))
	}
	return strings.Join(parts, "  ·  ")
}

// playButton is the accent button with a play glyph.
func (a *app) playButton(c *ui.Context, label string) bool {
	t := c.Theme()
	b := ui.ButtonBase(c.Key("play")).Padding(8, 20, 8, 16).Radius(9).Label(label).Cursor(ui.CursorPointer)
	bg := t.Accent
	if b.Pressed() {
		bg = t.AccentPressed
	} else if b.Hovered() {
		bg = t.AccentHover
	}
	b.Background(bg).TextColor(t.AccentText)
	b.Children(func() {
		ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconPlay).Size(13, 13)
			ui.Text(c, label).FontSize(14).FontWeight(600)
		})
	})
	return b.Clicked()
}

// favButton saves the series, or drops it when it is saved already.
func (a *app) favButton(c *ui.Context, sp *seriesPage) {
	t := c.Theme()
	saved := a.favs.has(sp.id)
	label := "收藏"
	if saved {
		label = "已收藏"
	}
	// The label names the action, not the state, and says which: the
	// sidebar has a 收藏 of its own.
	action := favLabel(saved) + "这部剧"
	b := ui.ButtonBase(c.Key("fav")).Padding(8, 18, 8, 15).Radius(9).Label(action).
		Tooltip(action).Cursor(ui.CursorPointer)
	switch {
	case b.Pressed():
		b.Background(t.Text.Alpha(0.2))
	case b.Hovered():
		b.Background(t.Text.Alpha(0.14))
	default:
		b.Background(t.Text.Alpha(0.09))
	}
	b.Children(func() {
		ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
			ic := iconStar
			if saved {
				ic = iconStarOn
			}
			ui.Icon(c, ic).Size(14, 14)
			ui.Text(c, label).FontSize(14).FontWeight(600)
		})
	})
	if b.Clicked() {
		// Saved as the grids show it, so the 收藏 grid matches the others.
		k := summaryCard(sp.series.SeriesSummary)
		k.ID = sp.id
		a.favs.toggle(favOf(k))
	}
}

func (a *app) episodeGrid(c *ui.Context, sp *seriesPage) {
	t := c.Theme()
	eps := sp.episodes
	cur := 0
	if w, ok := a.hist.get(sp.id); ok {
		cur = w.Episode
	}
	ui.Column(c).Padding(6, 44, 44, 44).Gap(16).Children(func() {
		// The other seasons, in the same place: Back still leaves the series.
		a.seasonPicker(c, sp, func(se model.Season) { a.router.Replace("/series/" + se.SeriesID) })
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "选集").FontSize(19).Bold()
			ui.Textf(c, "共 %d 集", len(eps)).FontSize(13).TextColor(t.TextMuted)
			ui.Spacer(c)
			a.partPicker(c, sp)
		})
		if len(eps) == 0 {
			ui.Text(c, "暂时没有可看的剧集").FontSize(13).TextColor(t.TextMuted)
			return
		}
		lo, hi := a.partRange(sp)
		ui.Row(c).Wrap().Gap(8).Children(func() {
			for _, e := range eps[lo:hi] {
				if a.episodeChip(c, e, e.Number == cur, t).Clicked() && e.Playable {
					a.play(sp.id, e.Number, 0)
				}
			}
		})
	})
}

// seasonPicker shows the seasons of a series in several, this one
// chosen; seasons still to come show, but can't be chosen yet.
func (a *app) seasonPicker(c *ui.Context, sp *seriesPage, choose func(model.Season)) {
	if len(sp.seasons) < 2 {
		return
	}
	t := c.Theme()
	ui.ScrollHorizontal(c.Key("seasons")).Scrollbars(false).Shrink(0).Children(func() {
		ui.Row(c).Gap(4).Children(func() {
			for _, se := range sp.seasons {
				if se.EpisodeCount == 0 {
					ui.Text(c.Key("season:"+se.SeriesID), seasonLabel(se)).FontSize(13).FontWeight(500).
						Padding(5, 12).TextColor(t.TextMuted.Alpha(0.6)).Tooltip("即将上线").SingleLine()
					continue
				}
				if a.chip(c, "season:"+se.SeriesID, seasonLabel(se), se.SeriesID == sp.id, t) {
					choose(se)
				}
			}
		})
	})
}

// playSeason plays a season from where the viewer left it, else from its
// first episode.
func (a *app) playSeason(id string) {
	if sp := a.series[id]; sp != nil && sp.series != nil {
		sid, n, start, _ := a.resumePoint(sp)
		a.play(sid, n, start)
		return
	}
	if w, ok := a.hist.get(id); ok && !w.Finished {
		a.play(id, w.Episode, max(0, w.Position-3))
		return
	}
	a.play(id, 1, 0)
}

// partPicker chooses which hundred episodes show, for long series.
func (a *app) partPicker(c *ui.Context, sp *seriesPage) {
	n := len(sp.episodes)
	if n <= episodesPerPart {
		return
	}
	var labels []string
	for lo := 0; lo < n; lo += episodesPerPart {
		labels = append(labels, fmt.Sprintf("%d–%d", lo+1, min(n, lo+episodesPerPart)))
	}
	if len(labels) <= 6 {
		ui.Segmented(c.Key("parts"), &sp.part, labels...)
		return
	}
	chosen := labels[min(sp.part, len(labels)-1)]
	ui.Select(c.Key("parts"), &chosen, labels).Width(130)
	for i, l := range labels {
		if l == chosen {
			sp.part = i
		}
	}
}

func (a *app) partRange(sp *seriesPage) (int, int) {
	n := len(sp.episodes)
	part := min(sp.part, max(0, (n-1)/episodesPerPart))
	return part * episodesPerPart, min(n, (part+1)*episodesPerPart)
}

// episodeChip is an episode's number to click, filled while it is the
// one the viewer is on.
func (a *app) episodeChip(c *ui.Context, e model.Episode, current bool, t *ui.Theme) ui.Element {
	b := ui.ButtonBase(c.Key(e.ID)).Size(56, 38).Radius(9).Center().Label(fmt.Sprintf("第 %d 集", e.Number))
	switch {
	case current:
		b.Background(t.Accent).TextColor(t.AccentText)
	case b.Hovered() && e.Playable:
		b.Background(t.Text.Alpha(0.12))
	default:
		b.Background(t.Text.Alpha(0.06))
	}
	if e.Playable {
		b.Cursor(ui.CursorPointer)
	} else {
		b.Opacity(0.45).Tooltip("这一集需要登录")
	}
	b.Children(func() {
		if e.Playable {
			ui.Textf(c, "%d", e.Number).FontSize(14).FontWeight(500).FontFeatures("tnum")
		} else {
			ui.Icon(c, iconLock).Size(13, 13)
		}
	})
	return b
}

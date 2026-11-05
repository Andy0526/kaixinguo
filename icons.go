package main

import "github.com/egoist/mygo/ui"

func icon(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}

var (
	iconHome     = icon(`<path d="M2.5 7.3 8 2.8l5.5 4.5v5.7a.8.8 0 0 1-.8.8H10v-3.6H6v3.6H3.3a.8.8 0 0 1-.8-.8z"/>`)
	iconChart    = icon(`<path d="M3.5 13.5v-5M8 13.5v-11M12.5 13.5v-7.5"/>`)
	iconCalendar = icon(`<rect x="2" y="3.2" width="12" height="10.8" rx="2.2"/><path d="M2 6.8h12M5.3 1.8v2.6M10.7 1.8v2.6"/>`)
	iconClock    = icon(`<circle cx="8" cy="8" r="6"/><path d="M8 4.9V8l2.1 1.5"/>`)
	iconBack     = icon(`<path d="M9.8 3.2 5 8l4.8 4.8" stroke-width="1.7"/>`)
	iconPlay     = icon(`<path d="M5 3.1v9.8a.6.6 0 0 0 .92.5l7.7-4.9a.6.6 0 0 0 0-1L5.92 2.6A.6.6 0 0 0 5 3.1z" fill="currentColor" stroke="none"/>`)
	iconLock     = icon(`<rect x="3.5" y="7" width="9" height="7" rx="1.6"/><path d="M5.5 7V5.2a2.5 2.5 0 0 1 5 0V7"/>`)
	iconStar     = icon(`<path d="M8 2.1 9.53 5.9l4.08.28-3.14 2.62 1 3.97L8 10.6l-3.47 2.17 1-3.97L2.39 6.18l4.08-.28z"/>`)
	iconStarOn   = icon(`<path d="M8 2.1 9.53 5.9l4.08.28-3.14 2.62 1 3.97L8 10.6l-3.47 2.17 1-3.97L2.39 6.18l4.08-.28z" fill="currentColor"/>`)
	iconFlame    = icon(`<path d="M8 14c2.6 0 4.5-1.8 4.5-4.4 0-2.6-1.9-3.9-2.6-6.1-.3 1.4-1 2.2-1.7 2.6C8.2 4.3 7.3 2.8 5.8 2c.3 2.3-2.3 3.9-2.3 7.6C3.5 12.2 5.4 14 8 14z"/>`)
)

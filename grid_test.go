package main

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestGridStaysAtTop(t *testing.T) {
	a := testApp(t)
	a.home = newFeed(nil)
	a.home.started, a.home.more = true, true
	a.home.loading = true
	tt := ui.NewTester(a.view, 1180, 760)
	tt.Frame()
	var cards []card
	for i := 0; i < 24; i++ {
		cards = append(cards, card{ID: fmt.Sprint(i), Title: fmt.Sprintf("剧%02d", i), Sub: "1 集"})
	}
	a.home.loading = false
	a.home.add(page{cards: cards, more: true})
	a.home.loading = true // as loadMore would be
	tt.Frame()
	tt.Frame()
	r0, _ := tt.Find("剧00")
	t.Logf("after the first page: %v", r0)
	var more []card
	for i := 24; i < 48; i++ {
		more = append(more, card{ID: fmt.Sprint(i), Title: fmt.Sprintf("剧%02d", i), Sub: "1 集"})
	}
	a.home.loading = false
	a.home.add(page{cards: more, more: true})
	tt.Frame()
	tt.Frame()
	r1, ok := tt.Find("剧00")
	t.Logf("after the second page: %v %v", r1, ok)
	if !ok || r1.Y != r0.Y {
		t.Fatalf("the grid moved from %v to %v", r0, r1)
	}
}

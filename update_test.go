package main

import (
	"fmt"
	"strings"
	"testing"
)

// A newer version shows at the sidebar's foot until it is gone, and a
// click on it shows the update window again.
func TestUpdateBanner(t *testing.T) {
	a, tt := sidebarApp(t)
	if tt.HasText("新版本 0.4.0") {
		t.Fatal("a new version shows before any check found one")
	}
	checks := 0
	a.checkUpdates = func() { checks++ }
	a.newVersion = "0.4.0"
	tt.Frame()
	click(t, tt, "新版本 0.4.0")
	if checks != 1 {
		t.Fatalf("%d checks after a click on the new version", checks)
	}
	a.newVersion = "" // installed or skipped
	tt.Frame()
	if tt.HasText("新版本 0.4.0") {
		t.Fatal("the new version still shows once it is gone")
	}
}

// The update window's texts fill with their arguments, as mygo.Use checks
// at launch, panicking otherwise.
func TestUpdaterStrings(t *testing.T) {
	s := updaterOptions.Strings[updaterOptions.Language]
	for text, n := range map[string]int{s.Available: 1, s.UnavailableMessage: 1, s.ReadyMessage: 2} {
		args := make([]any, n)
		for i := range args {
			args[i] = "x"
		}
		if out := fmt.Sprintf(text, args...); strings.Contains(out, "%!") {
			t.Errorf("%q does not fill with %d arguments: %q", text, n, out)
		}
	}
}

//go:build darwin

package main

import (
	"time"

	"github.com/ebitengine/purego/objc"
)

// doubleClickInterval is the system's double-click window. Paying
// attention to it keeps a click on the picture from pausing before a
// slower double click asks for the theater.
func doubleClickInterval() time.Duration {
	class := objc.GetClass("NSEvent")
	if class == 0 {
		return 500 * time.Millisecond
	}
	seconds := objc.Send[float64](objc.ID(class), objc.RegisterName("doubleClickInterval"))
	if seconds <= 0 || seconds > 5 {
		return 500 * time.Millisecond
	}
	return time.Duration(seconds * float64(time.Second))
}

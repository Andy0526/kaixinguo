//go:build !darwin

package main

import "time"

func doubleClickInterval() time.Duration { return 500 * time.Millisecond }

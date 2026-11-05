//go:build e2e

package main

// Builds with -tags e2e can be driven from outside through the socket
// MYGO_AUTOMATION names: clicked, read and captured, as end-to-end tests do.
import _ "github.com/egoist/mygo/plugins/automation"

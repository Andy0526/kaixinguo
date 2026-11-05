package main

import (
	"encoding/json"
	"os"
	"sync"
)

// store keeps a list of values in a JSON file. The list itself belongs to
// the main thread, which owns it; store only writes snapshots of it, on a
// goroutine of its own, so saving never holds up a frame.
type store[T any] struct {
	path    string
	save    chan []T
	pending sync.WaitGroup // snapshots put and not yet written or dropped
}

// openStore reads the list in the file at path and starts the goroutine
// that writes it. A file that is missing or unreadable reads as an empty
// list: the app opens clean rather than refusing to open.
func openStore[T any](path string) (*store[T], []T) {
	s := &store[T]{path: path, save: make(chan []T, 1)}
	var list []T
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &list)
	}
	go s.writer()
	return s, list
}

// put saves list. A snapshot still waiting to be written is dropped:
// only the latest matters. Only the main thread puts.
func (s *store[T]) put(list []T) {
	snapshot := append([]T(nil), list...)
	s.pending.Add(1)
	select {
	case <-s.save:
		s.pending.Done() // the one this replaces
	default:
	}
	s.save <- snapshot
}

// flush waits until every list put so far is on disk, as the app quits.
func (s *store[T]) flush() { s.pending.Wait() }

func (s *store[T]) writer() {
	for list := range s.save {
		s.write(list)
		s.pending.Done()
	}
}

func (s *store[T]) write(list []T) {
	data, err := json.Marshal(list)
	if err != nil {
		return
	}
	// Write beside the file and move it over, so a crash halfway through
	// cannot leave half a file behind.
	tmp := s.path + ".tmp"
	if os.WriteFile(tmp, data, 0o644) == nil {
		_ = os.Rename(tmp, s.path)
	}
}

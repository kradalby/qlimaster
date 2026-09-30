package ui

import (
	"errors"
	"sync"
)

// saver orders writes of numbered snapshots to one file. bubbletea runs
// every command on its own goroutine, so without it an older snapshot's
// write can finish after a newer one and win.
type saver[T any] struct {
	write func(T) error

	mu      sync.Mutex
	written uint64 // seq of the newest snapshot on disk
}

func newSaver[T any](write func(T) error) *saver[T] {
	return &saver[T]{write: write}
}

// save writes v, snapshot seq, unless a newer snapshot is already on disk.
func (s *saver[T]) save(seq uint64, v T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if seq <= s.written {
		return nil
	}

	if err := s.write(v); err != nil {
		return err
	}

	s.written = seq

	return nil
}

// Flush synchronously writes the quiz and history snapshots not yet on
// disk. bubbletea does not wait for running commands on quit, so callers
// flush the model returned by Program.Run.
func (m Model) Flush() error {
	return errors.Join(
		m.quizSaver.save(m.quizSeq, m.quiz),
		m.historySaver.save(m.historySeq, m.history),
	)
}

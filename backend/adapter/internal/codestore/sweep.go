package codestore

import (
	"context"
	"log/slog"
	"time"
)

// Records is what the sweep asks of B's database (internal/store).
type Records interface {
	// UnreferencedCode returns the keys among keys that no function_code row
	// names.
	UnreferencedCode(ctx context.Context, keys []string) ([]string, error)
	// DeleteUnreferencedCode calls del under the node's lock when no row
	// names key, and reports whether it did.
	DeleteUnreferencedCode(ctx context.Context, diagramID, nodeID, key string, del func(context.Context) error) (bool, error)
}

const (
	// SweepEvery is how often B API sweeps the code store.
	SweepEvery = time.Hour
	// sweepPerPass caps how many keys one pass reads.
	sweepPerPass = 1000
)

// Sweeper deletes code objects that no function_code row names. B deletes an
// object itself when a user removes or replaces a function's code; the rows
// a delete in A takes away (a diagram's, a project's or an organization's,
// by ON DELETE CASCADE) leave their objects behind, and only the sweep finds
// those. It also finds an object whose own delete failed.
//
// Each pass reads at most PerPass keys, starting after the last key the
// previous pass read, and starts again at the first key after the store's
// end, so a large store is swept in turns. Its log lines name counts and ids,
// never content.
type Sweeper struct {
	Objects Objects
	Records Records
	Logger  *slog.Logger
	// PerPass caps how many keys one pass reads; 0 means 1000.
	PerPass int

	after string
}

// Pass sweeps the next keys once and returns how many objects it deleted.
// An object it could not delete is logged and left for a later pass; the
// error is the first such failure, or a failure to list or to ask the
// database.
func (s *Sweeper) Pass(ctx context.Context) (int, error) {
	per := s.PerPass
	if per <= 0 {
		per = sweepPerPass
	}
	keys, err := s.Objects.List(ctx, s.after, per)
	if err != nil {
		return 0, err
	}
	if len(keys) < per {
		s.after = ""
	} else {
		s.after = keys[len(keys)-1]
	}
	if len(keys) == 0 {
		return 0, nil
	}
	unused, err := s.Records.UnreferencedCode(ctx, keys)
	if err != nil {
		return 0, err
	}
	deleted := 0
	var first error
	for _, key := range unused {
		diagramID, nodeID, ok := KeyNode(key)
		if !ok {
			continue
		}
		done, err := s.Records.DeleteUnreferencedCode(ctx, diagramID, nodeID, key, func(ctx context.Context) error {
			return s.Objects.Delete(ctx, key)
		})
		if err != nil {
			s.logger().Error("could not remove function code no record names", "diagram_id", diagramID, "node_id", nodeID, "err", err)
			if first == nil {
				first = err
			}
			continue
		}
		if done {
			deleted++
		}
	}
	if deleted > 0 {
		s.logger().Info("removed function code no record names", "objects", deleted)
	}
	return deleted, first
}

// Run sweeps right away and then every interval, until ctx ends.
func (s *Sweeper) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		if _, err := s.Pass(ctx); err != nil && ctx.Err() == nil {
			s.logger().Error("function code sweep", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (s *Sweeper) logger() *slog.Logger {
	if s.Logger == nil {
		return slog.Default()
	}
	return s.Logger
}

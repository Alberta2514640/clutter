package terraform

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
)

// MaxLine caps one line a LineWriter holds while it waits for the line's end.
const MaxLine = 64 << 10

// LongLine is what a LineWriter hands on in place of a line over MaxLine.
const LongLine = "[Clutter left out a line of over 64 KB.]\n"

// LineWriter turns writes into whole lines and hands each, with its newline,
// to its function, in order. A line longer than MaxLine is left out whole and
// LongLine is handed on instead: cutting it could split a secret a later step
// must find whole. It is safe for concurrent use.
type LineWriter struct {
	mu   sync.Mutex
	line func(string) error
	buf  bytes.Buffer
	// skipping is true while the rest of a line over MaxLine is dropped.
	skipping bool
}

// NewLineWriter returns a LineWriter that hands each line to line. An error
// from line ends the write that caused it.
func NewLineWriter(line func(string) error) *LineWriter {
	return &LineWriter{line: line}
}

func (w *LineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := len(p)
	for len(p) > 0 {
		i := bytes.IndexByte(p, '\n')
		if i < 0 {
			if !w.skipping {
				w.buf.Write(p)
				if w.buf.Len() > MaxLine {
					w.buf.Reset()
					w.skipping = true
					if err := w.line(LongLine); err != nil {
						return 0, err
					}
				}
			}
			break
		}
		part := p[:i+1]
		p = p[i+1:]
		if w.skipping {
			w.skipping = false
			continue
		}
		if w.buf.Len()+len(part) > MaxLine+1 {
			w.buf.Reset()
			if err := w.line(LongLine); err != nil {
				return 0, err
			}
			continue
		}
		w.buf.Write(part)
		s := w.buf.String()
		w.buf.Reset()
		if err := w.line(s); err != nil {
			return 0, err
		}
	}
	return n, nil
}

// Flush hands on a last line that has no newline.
func (w *LineWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.skipping = false
	if w.buf.Len() == 0 {
		return nil
	}
	s := w.buf.String()
	w.buf.Reset()
	return w.line(s)
}

// planCounts is the counts of a plan summary. A summary may also count
// imports ("Plan: 1 to import, 2 to add, ..."), which it skips.
var planCounts = regexp.MustCompile(`(\d+) to add, (\d+) to change, (\d+) to destroy`)

// PlanSummary reads one line of Terraform's output: a plan's summary line
// ("Plan: 2 to add, 0 to change, 1 to destroy.") or its "No changes." line.
// ok is false for any other line. Only a line that starts with the summary
// counts, so a resource's own text cannot pass for one.
func PlanSummary(line string) (_ connectors.PlanResult, ok bool) {
	t := strings.TrimSpace(line)
	if strings.HasPrefix(t, "No changes.") {
		return connectors.PlanResult{}, true
	}
	if !strings.HasPrefix(t, "Plan:") {
		return connectors.PlanResult{}, false
	}
	m := planCounts.FindStringSubmatch(t)
	if m == nil {
		return connectors.PlanResult{}, false
	}
	add, _ := strconv.Atoi(m[1])
	change, _ := strconv.Atoi(m[2])
	destroy, _ := strconv.Atoi(m[3])
	return connectors.PlanResult{Add: add, Change: change, Destroy: destroy}, true
}

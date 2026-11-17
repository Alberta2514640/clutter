package jobs

import (
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
)

// redactor replaces credential values in text. Longer values are replaced
// first, so a value that contains another is hidden whole.
type redactor struct{ values []string }

func newRedactor(values []string) redactor {
	vs := append([]string(nil), values...)
	sort.Slice(vs, func(i, j int) bool { return len(vs[i]) > len(vs[j]) })
	return redactor{values: vs}
}

// with returns a redactor that also replaces values.
func (r redactor) with(values []string) redactor {
	return newRedactor(append(append([]string(nil), r.values...), values...))
}

func (r redactor) String(s string) string {
	for _, v := range r.values {
		s = strings.ReplaceAll(s, v, connectors.Redacted)
	}
	return s
}

// redactWriter writes whole lines to w with credential values removed
// (terraform.LineWriter). A value split across two writes is still caught,
// unless it spans a newline.
type redactWriter struct {
	mu    sync.Mutex
	w     io.Writer
	r     redactor
	lines *terraform.LineWriter
}

func newRedactWriter(w io.Writer) *redactWriter {
	rw := &redactWriter{w: w}
	rw.lines = terraform.NewLineWriter(func(line string) error {
		_, err := io.WriteString(rw.w, rw.redact(line))
		return err
	})
	return rw
}

func (rw *redactWriter) Write(p []byte) (int, error) { return rw.lines.Write(p) }

// set replaces the values the writer removes.
func (rw *redactWriter) set(values []string) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.r = newRedactor(values)
}

// add removes these values too from every line written after the call.
func (rw *redactWriter) add(values []string) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.r = rw.r.with(values)
}

// redact removes every value the writer knows from s.
func (rw *redactWriter) redact(s string) string {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	return rw.r.String(s)
}

// flush writes any partial last line.
func (rw *redactWriter) flush() error { return rw.lines.Flush() }

package aws

import (
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/Alberta2514640/clutter/backend/adapter/connectors"
	"github.com/Alberta2514640/clutter/backend/adapter/internal/terraform"
)

// runLog passes Terraform's output to the run's log line by line
// (terraform.LineWriter), removes the session keys this connector derived (the
// job runner removes only the connection's own values), and reads the plan
// summary (terraform.PlanSummary) and the ids of what an apply built.
type runLog struct {
	mu      sync.Mutex
	w       io.Writer
	secrets []string
	lines   *terraform.LineWriter

	plan     connectors.PlanResult
	planSeen bool
	created  map[string]string // Terraform address -> cloud id, for what the apply created or changed
}

func newRunLog(w io.Writer, keys SessionKeys) *runLog {
	var secrets []string
	for _, v := range keys.values() {
		if v != "" {
			secrets = append(secrets, v)
		}
	}
	sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
	l := &runLog{w: w, secrets: secrets, created: map[string]string{}}
	l.lines = terraform.NewLineWriter(l.line)
	return l
}

func (l *runLog) Write(p []byte) (int, error) { return l.lines.Write(p) }

// flush writes a last line that has no newline.
func (l *runLog) flush() error { return l.lines.Flush() }

func (l *runLog) line(s string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, v := range l.secrets {
		s = strings.ReplaceAll(s, v, connectors.Redacted)
	}
	if res, ok := terraform.PlanSummary(s); ok {
		l.plan, l.planSeen = res, true
	}
	if addr, id, ok := terraform.ChangedResource(s); ok {
		l.created[addr] = id
	}
	_, err := io.WriteString(l.w, s)
	return err
}

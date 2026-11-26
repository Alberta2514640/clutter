package connectors

import (
	"sort"
	"strings"
)

// DuplicateNames is the one name rule every connector applies: each block a
// deployment includes needs a name no other included block has, whatever its
// kind, with - and _ counted as the same character (some clouds' names take
// only one of them, and an environment variable named after a block turns
// both into _). It returns one problem on each block after the first that
// shares a name. Names already passed the capability check's pattern.
func DuplicateNames(g Graph, d Deployment) []Problem {
	ids := make([]string, 0, len(g.Blocks))
	for id := range g.Blocks {
		if !d.Excluded(id) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	var out []Problem
	seen := map[string]bool{}
	for _, id := range ids {
		key := strings.ReplaceAll(g.Blocks[id].Name, "_", "-")
		if seen[key] {
			out = append(out, Problem{NodeID: id, Field: "name", Code: CodeDuplicateName,
				Message: "Another block has the same name. Each block needs its own name (- and _ count as the same)."})
			continue
		}
		seen[key] = true
	}
	return out
}

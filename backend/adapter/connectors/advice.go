package connectors

import "sort"

// MsgClosedPublicVM is CodeClosedPublicVM's message.
const MsgClosedPublicVM = "This VM is public but its firewall allows nothing in."

// The portable fields ClosedPublicVMs reads (StandardField panics on a typo).
var (
	adviceIngress = StandardField(KindFirewall, "ingress").Name
	advicePublic  = StandardField(KindVM, "public").Name
)

// ClosedPublicVMs is the note every connector's Advisor gives about a public
// VM that nothing can reach: no firewall it is linked to lets any traffic in
// (it has none, or each has an empty ingress list). Every cloud Clutter
// builds on then lets nothing in from outside, so the public address answers
// nothing. Blocks d leaves out are skipped, and a firewall whose ingress does
// not parse is counted as open, since Validate already reports it.
func ClosedPublicVMs(g Graph, d Deployment) []Problem {
	open := map[string]bool{}
	for _, e := range g.Edges {
		fw, okF := g.Blocks[e.Source]
		vm, okV := g.Blocks[e.Target]
		if !okF || !okV || fw.Kind != KindFirewall || vm.Kind != KindVM || d.Excluded(fw.ID) || d.Excluded(vm.ID) {
			continue
		}
		ingress, _ := fw.Fields[adviceIngress].(string)
		rules, err := ParseIngress(ingress)
		if err != nil || len(rules) > 0 {
			open[vm.ID] = true
		}
	}
	var out []Problem
	for id, b := range g.Blocks {
		if b.Kind != KindVM || d.Excluded(id) || open[id] {
			continue
		}
		if public, _ := b.Fields[advicePublic].(bool); public {
			out = append(out, Problem{NodeID: id, Code: CodeClosedPublicVM, Message: MsgClosedPublicVM})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NodeID < out[j].NodeID })
	return out
}

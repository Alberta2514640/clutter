package connectors

import (
	"encoding/json"
	"regexp"
	"strings"
)

// The results an apply returns about what it built, by name. The deploy
// panel shows them under "What was built": a server's addresses, an HTTP
// API's address, a bucket's or a table's name. Each is a plain value, never a
// secret.
const (
	ResultPrivateAddress = "private_address"
	ResultPublicAddress  = "public_address"
	ResultURL            = "url"
	ResultBucket         = "bucket"
	ResultTable          = "table"
	// ResultCode is what a function runs. Its one value, ResultCodeSample,
	// says the function still runs Clutter's sample code, which answers every
	// request, because no code was uploaded for it.
	ResultCode = "code"
	// ResultSimulated says a VM was built on a cloud emulator (LocalStack): it
	// has addresses but never boots, so nothing on it can be opened. Its one
	// value is ResultSimulatedEmulator. A connector gives it only when it
	// knows its cloud is an emulator.
	ResultSimulated = "simulated"
)

// ResultCodeSample is ResultCode's value for a function that runs the sample.
const ResultCodeSample = "sample"

// ResultSimulatedEmulator is ResultSimulated's value.
const ResultSimulatedEmulator = "emulator"

// ResultNames are the results a block of a kind can have, in the order the
// deploy panel lists them. A connector renders an output for each one it can
// give (ResultOutputName) and reads them back with ReadResults.
func ResultNames(kind BlockKind) []string {
	switch kind {
	case KindVM:
		return []string{ResultPrivateAddress, ResultPublicAddress, ResultSimulated}
	case KindHTTPAPI:
		return []string{ResultURL}
	case KindBucket:
		return []string{ResultBucket}
	case KindTable:
		return []string{ResultTable}
	case KindFunction:
		return []string{ResultCode}
	}
	return nil
}

// Result is one value an apply returns about one block.
type Result struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// resultNodeID is translate.NodeIDPattern: letters, digits, hyphens and
// underscores, so an output name built from it is a Terraform identifier.
var resultNodeID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ResultOutputName is the Terraform output that carries one result of one
// block. ok is false for a node id translate would refuse: such an id never
// reaches rendered Terraform, so its block has no results.
func ResultOutputName(nodeID, name string) (_ string, ok bool) {
	if !resultNodeID.MatchString(nodeID) {
		return "", false
	}
	return "result_" + name + "_" + nodeID, true
}

// ReadResults reads each included block's results from a run's `terraform
// output -json`. A result whose output is missing, null, sensitive, not text
// or not a value CheckCrossValue accepts is left out: results only inform the
// user, so a bad one never fails the run. Blocks with no result are left out.
func ReadResults(raw []byte, g Graph, d Deployment) map[string][]Result {
	all := map[string]outputJSON{}
	if len(strings.TrimSpace(string(raw))) == 0 || json.Unmarshal(raw, &all) != nil {
		return map[string][]Result{}
	}
	out := map[string][]Result{}
	for id, b := range g.Blocks {
		if d.Excluded(id) {
			continue
		}
		for _, name := range ResultNames(b.Kind) {
			key, ok := ResultOutputName(id, name)
			if !ok {
				continue
			}
			o, ok := all[key]
			if !ok || o.Sensitive {
				continue
			}
			var s string
			if json.Unmarshal(o.Value, &s) != nil || s == "" || CheckCrossValue(s) != nil {
				continue
			}
			out[id] = append(out[id], Result{Name: name, Value: s})
		}
	}
	return out
}

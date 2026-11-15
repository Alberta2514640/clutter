package connectors

import (
	"reflect"
	"testing"
)

func TestResultNames(t *testing.T) {
	if got := ResultNames(KindVM); !reflect.DeepEqual(got, []string{ResultPrivateAddress, ResultPublicAddress}) {
		t.Fatalf("vm results = %v", got)
	}
	if ResultNames(KindVolume) != nil {
		t.Fatal("a volume has no results")
	}
	if got, ok := ResultOutputName("web-1", ResultURL); !ok || got != "result_url_web-1" {
		t.Fatalf("output name = %q, %v", got, ok)
	}
	if _, ok := ResultOutputName(`x" { }`, ResultURL); ok {
		t.Fatal("a node id translate refuses named an output")
	}
}

func TestReadResults(t *testing.T) {
	g := Graph{Blocks: map[string]Block{
		"web":   {ID: "web", Kind: KindVM},
		"api":   {ID: "api", Kind: KindHTTPAPI},
		"files": {ID: "files", Kind: KindBucket},
		"gone":  {ID: "gone", Kind: KindTable},
		"bad":   {ID: "bad", Kind: KindTable},
	}}
	raw := []byte(`{
		"result_private_address_web": {"sensitive": false, "value": "10.0.0.5"},
		"result_public_address_web":  {"sensitive": false, "value": null},
		"result_url_api":             {"sensitive": false, "value": "https://abc.execute-api.ca-central-1.amazonaws.com/v1/items"},
		"result_bucket_files":        {"sensitive": true, "value": "c1-files"},
		"result_table_gone":          {"sensitive": false, "value": "c1-gone"},
		"result_table_bad":           {"sensitive": false, "value": "has space'"},
		"id_web":                     {"sensitive": false, "value": "i-1"}
	}`)
	got := ReadResults(raw, g, Deployment{Exclusions: []string{"gone"}})
	want := map[string][]Result{
		"web": {{Name: ResultPrivateAddress, Value: "10.0.0.5"}},
		"api": {{Name: ResultURL, Value: "https://abc.execute-api.ca-central-1.amazonaws.com/v1/items"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadResults = %+v, want %+v", got, want)
	}
	if got := ReadResults([]byte("not json"), g, Deployment{}); len(got) != 0 {
		t.Fatalf("unreadable outputs gave %+v", got)
	}
	if got := ReadResults(nil, g, Deployment{}); len(got) != 0 {
		t.Fatalf("no outputs gave %+v", got)
	}
}

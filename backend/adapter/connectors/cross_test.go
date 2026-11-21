package connectors

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestCrossClassFollowsTheEdgeTypes(t *testing.T) {
	cases := []struct {
		s, t BlockKind
		want EdgeClass
	}{
		{KindVM, KindVM, EdgeAddress},
		{KindVM, KindBucket, EdgePermission},
		{KindVM, KindTable, EdgePermission},
		// One cloud's wiring never crosses.
		{KindVM, KindNetwork, ""},
		{KindVolume, KindVM, ""},
		{KindFirewall, KindVM, ""},
		// Inside one cloud's IAM.
		{KindFunction, KindBucket, ""},
		{KindFunction, KindTable, ""},
		{KindHTTPAPI, KindFunction, ""},
	}
	for _, c := range cases {
		if got := CrossClass(c.s, c.t); got != c.want {
			t.Errorf("CrossClass(%s, %s) = %q, want %q", c.s, c.t, got, c.want)
		}
		if c.want != "" && len(ExportValues(c.want, c.t)) == 0 {
			t.Errorf("a %s edge to a %s carries no values", c.want, c.t)
		}
	}
	if !SensitiveValue(ValueSecretAccessKey) || SensitiveValue(ValueAccessKeyID) || SensitiveValue(ValueAddress) {
		t.Fatal("only the secret access key is sensitive")
	}
}

func TestCrossKeyIsStableAndSafe(t *testing.T) {
	k := CrossKey("n1", "n2")
	if k != CrossKey("n1", "n2") || k == CrossKey("n2", "n1") || !ValidCrossKey(k) {
		t.Fatalf("CrossKey gave %q", k)
	}
	if ValidCrossKey("x0123") || ValidCrossKey("y0123456789") {
		t.Fatal("a malformed key passes")
	}
}

func TestASensitiveValueNeverPrints(t *testing.T) {
	v := Value{Text: "s3cr3t-value", Sensitive: true}
	b, _ := json.Marshal(map[string]Value{"k": v})
	for _, s := range []string{v.String(), fmt.Sprintf("%v %+v %#v", v, v, v), string(b)} {
		if strings.Contains(s, "s3cr3t") {
			t.Fatalf("printed %q", s)
		}
	}
	if (Value{Text: "1.2.3.4"}).String() != "1.2.3.4" {
		t.Fatal("a plain value is hidden")
	}
	out := Outputs{"x": {"a": {Text: "plain"}, "b": v}}
	if got := out.SecretValues(); len(got) != 1 || got[0] != "s3cr3t-value" {
		t.Fatalf("SecretValues = %v", got)
	}
}

// unquote evaluates the HCL string StartupHCL returns, for a test: it holds
// only escapes hclsafe writes and ${...} references, which stay as written.
func unquote(t *testing.T, expr string) string {
	t.Helper()
	if !strings.HasPrefix(expr, `"`) || !strings.HasSuffix(expr, `"`) {
		t.Fatalf("not a string: %s", expr)
	}
	s, err := strconv.Unquote(strings.ReplaceAll(strings.ReplaceAll(expr, "$${", "${"), "%%{", "%{"))
	if err != nil {
		t.Fatalf("unquote %s: %v", expr, err)
	}
	return s
}

func TestStartupHCLHandsTheValuesThenRunsTheScript(t *testing.T) {
	fields := map[string]any{FieldStartupScript: "echo hi"}
	expr, sensitive, err := StartupHCL(fields, []EnvValue{
		{Name: EnvName("db", ValueAddress), Ref: "aws_instance.c1_db.private_ip"},
		{Name: EnvName("files", ValueSecretAccessKey), Text: "abc/DEF+123=", Sensitive: true},
		{Name: EnvName("files", ValueBucket), Text: "c1-files-0a1b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sensitive {
		t.Fatal("a secret value did not make the user data sensitive")
	}
	text := unquote(t, expr)
	if !strings.HasPrefix(text, "#!/bin/sh\n") {
		t.Fatalf("no shell line: %q", text)
	}
	for _, want := range []string{
		"CLUTTER_DB_ADDRESS='${aws_instance.c1_db.private_ip}'\n",
		"CLUTTER_FILES_BUCKET='c1-files-0a1b'\n",
		"CLUTTER_FILES_SECRET_ACCESS_KEY='abc/DEF+123='\n",
		"umask 077",
		". /etc/clutter/connections.env",
		"exec /var/lib/clutter/startup",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("user data lacks %q:\n%s", want, text)
		}
	}
	// Names are sorted, so a render is stable.
	if strings.Index(text, "CLUTTER_DB_ADDRESS") > strings.Index(text, "CLUTTER_FILES_BUCKET") {
		t.Fatal("values are not in name order")
	}
	// The script reaches the text only as base64.
	b64, _ := StartupUserData(fields)
	if !strings.Contains(text, "echo '"+b64+"' | base64 -d") || strings.Contains(text, "echo hi") {
		t.Fatalf("the script is not passed as base64:\n%s", text)
	}
	raw, _ := base64.StdEncoding.DecodeString(b64)
	if string(raw) != "#!/bin/sh\necho hi" {
		t.Fatalf("script %q", raw)
	}
}

func TestStartupHCLRefusesWhatCouldBreakOut(t *testing.T) {
	for name, env := range map[string][]EnvValue{
		"quote":      {{Name: "CLUTTER_A_ADDRESS", Text: "1.2.3.4'; rm -rf /"}},
		"template":   {{Name: "CLUTTER_A_ADDRESS", Text: "${file(\"/etc/passwd\")}"}},
		"newline":    {{Name: "CLUTTER_A_ADDRESS", Text: "a\nb"}},
		"empty":      {{Name: "CLUTTER_A_ADDRESS", Text: ""}},
		"bad name":   {{Name: "clutter_a", Text: "x"}},
		"same names": {{Name: "CLUTTER_A_ADDRESS", Text: "x"}, {Name: "CLUTTER_A_ADDRESS", Text: "y"}},
	} {
		if _, _, err := StartupHCL(nil, env); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if expr, _, err := StartupHCL(map[string]any{FieldStartupScript: "x"}, nil); err != nil || expr != "" {
		t.Fatalf("no values gave %q, %v; want the plain script path", expr, err)
	}
	long := strings.Repeat("a", MaxTextField-100)
	if _, _, err := StartupHCL(map[string]any{FieldStartupScript: long}, []EnvValue{{Name: "CLUTTER_A_ADDRESS", Text: "1.2.3.4"}}); err == nil {
		t.Fatal("a wrapper over the user data limit was accepted")
	}
}

func TestParseCrossOutputs(t *testing.T) {
	exports := []CrossEdge{
		{Source: "vm", Target: "b", Class: EdgePermission, Key: CrossKey("vm", "b")},
		{Source: "vm", Target: "peer", Class: EdgeAddress, Key: CrossKey("vm", "peer")},
	}
	kinds := map[string]BlockKind{"b": KindBucket, "peer": KindVM}
	kb, kp := exports[0].Key, exports[1].Key
	full := fmt.Sprintf(`{
	  "%[1]s": {"sensitive": false, "type": "string", "value": "c1-files-0a1b"},
	  "%[2]s": {"sensitive": false, "type": "string", "value": "ca-central-1"},
	  "%[3]s": {"sensitive": false, "type": "string", "value": "AKIAEXAMPLE0000001"},
	  "%[4]s": {"sensitive": true, "type": "string", "value": "secret/Value+0123456789"},
	  "%[5]s": {"sensitive": false, "type": "string", "value": "198.51.100.7"},
	  "other": {"sensitive": false, "type": "string", "value": "ignored"}
	}`, CrossOutputName(kb, ValueBucket), CrossOutputName(kb, ValueRegion), CrossOutputName(kb, ValueAccessKeyID),
		CrossOutputName(kb, ValueSecretAccessKey), CrossOutputName(kp, ValueAddress))
	out, err := ParseCrossOutputs([]byte(full), exports, kinds, true)
	if err != nil {
		t.Fatal(err)
	}
	if out[kb][ValueSecretAccessKey].Text != "secret/Value+0123456789" || !out[kb][ValueSecretAccessKey].Sensitive {
		t.Fatalf("secret %v", out[kb][ValueSecretAccessKey])
	}
	if out[kp][ValueAddress].Text != "198.51.100.7" || out[kp][ValueAddress].Sensitive {
		t.Fatalf("address %v", out[kp][ValueAddress])
	}

	// A plan before the first apply has none of them, which is no error.
	if out, err := ParseCrossOutputs([]byte(`{}`), exports, kinds, false); err != nil || len(out) != 0 {
		t.Fatalf("empty outputs: %v, %v", out, err)
	}
	// An apply must return every value.
	if _, err := ParseCrossOutputs([]byte(`{}`), exports, kinds, true); err == nil {
		t.Fatal("missing values passed an apply")
	}
	// A value that could break out is refused, and the error never repeats it.
	bad := strings.Replace(full, "198.51.100.7", "1.2.3.4' && curl evil", 1)
	if _, err := ParseCrossOutputs([]byte(bad), exports, kinds, true); err == nil || strings.Contains(err.Error(), "evil") {
		t.Fatalf("a bad value: %v", err)
	}
	if _, err := ParseCrossOutputs([]byte(`not json`), exports, kinds, false); err == nil {
		t.Fatal("broken JSON passed")
	}
}

func TestAddressCycle(t *testing.T) {
	ids := []string{"a", "b", "c"}
	if c := AddressCycle(ids, map[string][]string{"a": {"b"}, "b": {"c"}}); c != "" {
		t.Fatalf("a chain is a cycle at %q", c)
	}
	if c := AddressCycle(ids, map[string][]string{"a": {"b"}, "b": {"c"}, "c": {"a"}}); c == "" {
		t.Fatal("a circle was missed")
	}
}

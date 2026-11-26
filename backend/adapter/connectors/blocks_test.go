package connectors

import (
	"strings"
	"testing"
)

// Throwaway public keys; their private halves were never kept.
const (
	testEd25519 = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAAaLzQkB8bkh7I4/uWuVxHsQDSjL4RIq6y+JnLJP5Nn"
	testRSA2048 = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDt4j8XS2qlOsDjgL8Ne4+C5GTkDpzwE1TdMNd3lXRqCGstT6PLCzCPRMUiHQzjJHW2dxraOuIHrLdxYo0XPoGfdjrsnzLqCm7Slho8GqS40v1JcBnHhdeJ3Q+5Dcf1i2L4DUkTTbGDWvz7URXRXdArumuizRRo7RebmGmqP7DqHX+gLZPevLNvd10WJ5ecVdXBh2IogoDT9wO7N7EH8c+02hcnAYUbdGbExIyuXtw7KcoxAlSk9fprKB7FqdbmUf8vroUzbJGI3ezYPDiOAAK2FDCBxMt5yAtjIaVrJBF4SbrJPnOJ0hPoa20OjOCjZu1+FK/iBd3oIjWtDfF3Ru43"
	testRSA1024 = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQDdMyvK+HMSG5/HiMThP6LDOlsOnGxFhf8vxXHmcnXBLbNq+S46BGQ5z9GSUa2cATKSttEJAnA4nWd75AusBoEawybqkU+VkY7e0Q4E1j0NiEo5bssUqMpKg/0hOsIRruPWA90NHsjoOvdwUSLoGSMmpifDQ8yqTxD6hxZ11rE2Fw=="
	testECDSA   = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHTaLzbWpqeCyVeRlTULZjefgtcG3Sh6FC5l166is9I2joZyaeaKYDcn1Eh1zRMjEvz5HyKnPMuHQ3MMGLGpP4M="
)

func TestSSHPublicKey(t *testing.T) {
	ok := map[string]string{
		testEd25519:                   testEd25519,
		"  " + testEd25519 + " me@x ": testEd25519, // the comment is dropped
		testRSA2048 + " ":             testRSA2048,
		"":                            "",
		"   ":                         "",
	}
	for in, want := range ok {
		got, err := SSHPublicKey(map[string]any{FieldSSHKey: in})
		if err != nil || got != want {
			t.Errorf("SSHPublicKey(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if got, err := SSHPublicKey(map[string]any{}); got != "" || err != nil {
		t.Errorf("no key: %q, %v", got, err)
	}

	ed := strings.Fields(testEd25519)[1]
	refused := map[string]any{
		"a short RSA key":     testRSA1024,
		"an ECDSA key":        testECDSA,
		"two lines":           testEd25519 + "\n" + testEd25519,
		"a type and no body":  "ssh-ed25519",
		"a body that is not":  "ssh-ed25519 not-base64!",
		"a type it is not":    "ssh-rsa " + ed,
		"a cut body":          "ssh-ed25519 " + ed[:20],
		"a private key":       "-----BEGIN OPENSSH PRIVATE KEY-----",
		"interpolation":       "ssh-ed25519 ${file(\"/etc/passwd\")}",
		"a quote in the body": `ssh-ed25519 AAAA"`,
		"not text":            42,
	}
	for name, in := range refused {
		if got, err := SSHPublicKey(map[string]any{FieldSSHKey: in}); err == nil {
			t.Errorf("%s: accepted as %q", name, got)
		}
	}
}

// A new firewall lets nothing in until the user lists what may.
func TestFirewallDefaultsToNoRules(t *testing.T) {
	f := StandardField(KindFirewall, "ingress")
	if f.Default != nil {
		t.Fatalf("the firewall's ingress defaults to %v; want no rules", f.Default)
	}
	if rules, err := ParseIngress(""); err != nil || len(rules) != 0 {
		t.Fatalf("an empty ingress field is %v, %v; want no rules", rules, err)
	}
}

func TestDuplicateNames(t *testing.T) {
	g := Graph{Blocks: map[string]Block{
		"a": {ID: "a", Kind: KindVM, Name: "web"},
		"b": {ID: "b", Kind: KindFirewall, Name: "web"},
		"c": {ID: "c", Kind: KindNetwork, Name: "app_net"},
		"d": {ID: "d", Kind: KindBucket, Name: "app-net"},
		"e": {ID: "e", Kind: KindVolume, Name: "data"},
	}}
	ps := DuplicateNames(g, Deployment{})
	if len(ps) != 2 || ps[0].NodeID != "b" || ps[1].NodeID != "d" || ps[0].Code != CodeDuplicateName {
		t.Fatalf("problems %+v; want one on b and one on d", ps)
	}
	// A block left out of the deployment takes no name.
	if ps := DuplicateNames(g, Deployment{Exclusions: []string{"a", "c"}}); len(ps) != 0 {
		t.Fatalf("problems with the first of each pair left out: %+v", ps)
	}
}

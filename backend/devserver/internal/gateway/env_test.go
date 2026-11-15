package gateway

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The log-in Lambda accepts only Google tokens minted for the client id it is
// given, so the devserver must give it the one the frontend signs in with.
// Locally that is the NEXT_PUBLIC_GOOGLE_CLIENT_ID dev.sh starts the frontend
// with, and the frontend must read that value rather than hold its own.
func TestGoogleClientIDMatchesTheFrontend(t *testing.T) {
	devSh, err := os.ReadFile(filepath.Join("..", "..", "dev.sh"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`NEXT_PUBLIC_GOOGLE_CLIENT_ID="([^"]+)"`).FindAllSubmatch(devSh, -1)
	if len(m) != 1 {
		t.Fatalf("dev.sh sets NEXT_PUBLIC_GOOGLE_CLIENT_ID %d times; want once", len(m))
	}
	if got := string(m[0][1]); got != FrontendGoogleClientID {
		t.Fatalf("dev.sh passes %s to the frontend, but the log-in Lambda gets %s; update FrontendGoogleClientID in env.go", got, FrontendGoogleClientID)
	}

	provider := filepath.Join("..", "..", "..", "..", "frontend", "clutter", "src", "providers", "GoogleAuthProvider.tsx")
	src, err := os.ReadFile(provider)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID") {
		t.Fatalf("%s does not read NEXT_PUBLIC_GOOGLE_CLIENT_ID", provider)
	}
	if strings.Contains(string(src), ".apps.googleusercontent.com") {
		t.Fatalf("%s holds a client id of its own; it must read NEXT_PUBLIC_GOOGLE_CLIENT_ID only", provider)
	}

	got, err := lambdaEnvValue("GOOGLE_CLIENT_ID", Config{})
	if err != nil || got != FrontendGoogleClientID {
		t.Fatalf("GOOGLE_CLIENT_ID = %q, %v", got, err)
	}
}

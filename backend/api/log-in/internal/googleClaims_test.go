package internal

import "testing"

func TestCheckGoogleToken(t *testing.T) {
	const client = "clutter-client.apps.googleusercontent.com"
	good := map[string]interface{}{"email": "a@example.com", "email_verified": true, "name": "A", "picture": "https://p"}

	cases := []struct {
		name     string
		clientID string
		audience string
		claims   map[string]interface{}
		wantErr  bool
	}{
		{"valid token", client, client, good, false},
		{"no client id configured", "", "", good, true},
		{"token minted for another app", client, "other-app.apps.googleusercontent.com", good, true},
		{"unverified email", client, client, map[string]interface{}{"email": "a@example.com", "email_verified": false}, true},
		{"email_verified missing", client, client, map[string]interface{}{"email": "a@example.com"}, true},
		{"email missing", client, client, map[string]interface{}{"email_verified": true}, true},
		{"email of the wrong type", client, client, map[string]interface{}{"email": 7, "email_verified": true}, true},
		{"name and picture missing", client, client, map[string]interface{}{"email": "a@example.com", "email_verified": true}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			user, err := CheckGoogleToken(c.clientID, c.audience, c.claims)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if err == nil && user.Email != "a@example.com" {
				t.Fatalf("email = %q", user.Email)
			}
		})
	}
}

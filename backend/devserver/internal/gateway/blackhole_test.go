package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNoAWSKeepsTheLatestCallsWithStableIndexes(t *testing.T) {
	n, srv, _ := endpoint(t, realS3Module)
	n.limit = 3
	for i := range 5 {
		request(t, "POST", srv.URL+"/", "{}", map[string]string{"X-Amz-Target": fmt.Sprintf("Op.%d", i)})
	}
	if len(n.calls) != 3 {
		t.Fatalf("kept %d calls, want 3", len(n.calls))
	}

	calls := n.CallsHandler()
	get := func(since int) (int, []string) {
		rec := httptest.NewRecorder()
		calls.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/dev/aws-calls?since=%d", since), nil))
		var out struct {
			Next  int      `json:"next"`
			Calls []string `json:"calls"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out.Next, out.Calls
	}
	for _, c := range []struct {
		since int
		want  []string
	}{
		{0, []string{"Op.2", "Op.3", "Op.4"}}, // older than the oldest kept
		{3, []string{"Op.3", "Op.4"}},
		{5, []string{}},
		{1 << 30, []string{}}, // the smoke test's "where are we now"
		{-1, []string{}},
	} {
		next, got := get(c.since)
		if next != 5 || !reflect.DeepEqual(got, c.want) {
			t.Errorf("since %d: next %d, calls %v; want 5, %v", c.since, next, got, c.want)
		}
	}
}

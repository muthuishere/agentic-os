package msg

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// envOf turns a map into the env lookup New expects, so a test never reads the
// real process environment — a developer with MESSENGER_SERVE_TOKEN exported
// would otherwise silently change what these tests assert.
func envOf(pairs map[string]string) func(string) string {
	return func(key string) string { return pairs[key] }
}

// capture records the one request a handler saw, so a test can assert on the
// wire format rather than on the client's internals.
type capture struct {
	method string
	path   string
	query  string
	auth   string
	ctype  string
	body   []byte
}

func serve(t *testing.T, status int, response string, seen *capture) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*seen = capture{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.RawQuery,
			auth:   r.Header.Get("Authorization"),
			ctype:  r.Header.Get("Content-Type"),
			body:   body,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestNewPrefersTheAgenticOSHubURL(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"aos variable wins", map[string]string{
			"AGENTIC_OS_MESSENGER_URL": "http://127.0.0.1:1",
			"MESSENGER_URL":            "http://127.0.0.1:2",
		}, "http://127.0.0.1:1"},
		{"falls back to the shared variable", map[string]string{
			"MESSENGER_URL": "http://127.0.0.1:2",
		}, "http://127.0.0.1:2"},
		{"defaults to the local hub", nil, DefaultBaseURL},
	}
	for _, tc := range cases {
		if got := New(envOf(tc.env)).BaseURL; got != tc.want {
			t.Errorf("%s: BaseURL = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestSendCarriesTheBearerTokenWhenTheEnvVarIsSet(t *testing.T) {
	var seen capture
	server := serve(t, 200, `{"ok":true,"id":"m-1"}`, &seen)

	client := New(envOf(map[string]string{
		"AGENTIC_OS_MESSENGER_URL": server.URL,
		TokenEnv:                   "s3cret-hub-token",
	}))
	if _, err := client.Send(context.Background(), SendRequest{Channel: "ops", Text: "hi"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if seen.auth != "Bearer s3cret-hub-token" {
		t.Fatalf("Authorization = %q; the hub rejects an unauthenticated call", seen.auth)
	}
}

// A hub started without a token rejects requests that carry an Authorization
// header it cannot verify, so an unset variable must mean no header at all —
// not an empty `Bearer `.
func TestNoAuthorizationHeaderWithoutAToken(t *testing.T) {
	var seen capture
	server := serve(t, 200, `{"ok":true}`, &seen)

	client := New(envOf(map[string]string{"AGENTIC_OS_MESSENGER_URL": server.URL}))
	if _, err := client.Health(context.Background()); err != nil {
		t.Fatalf("health: %v", err)
	}
	if seen.auth != "" {
		t.Fatalf("Authorization = %q, want the header to be absent", seen.auth)
	}
}

func TestSendPostsTheDocumentedBody(t *testing.T) {
	var seen capture
	server := serve(t, 200, `{"ok":true,"id":"m-42"}`, &seen)

	client := New(envOf(map[string]string{"AGENTIC_OS_MESSENGER_URL": server.URL}))
	res, err := client.Send(context.Background(), SendRequest{
		Channel: "ops",
		Text:    "build passed",
		To:      "thread-7",
		ReplyTo: "last",
		File:    "report.pdf",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if res.ID != "m-42" {
		t.Fatalf("id = %q; the provider id is what threads a reply", res.ID)
	}
	if seen.method != http.MethodPost || seen.path != "/send" {
		t.Fatalf("%s %s, want POST /send", seen.method, seen.path)
	}
	if seen.ctype != "application/json" {
		t.Fatalf("Content-Type = %q", seen.ctype)
	}

	var got map[string]any
	if err := json.Unmarshal(seen.body, &got); err != nil {
		t.Fatalf("body is not JSON: %v (%s)", err, seen.body)
	}
	want := map[string]any{
		"channel":  "ops",
		"text":     "build passed",
		"to":       "thread-7",
		"reply_to": "last",
		"file":     "report.pdf",
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("body[%q] = %v, want %v", key, got[key], value)
		}
	}
}

// Empty optional fields are tagged omitempty so the hub applies its own
// defaults; sending "" for a thread id is not the same as sending nothing.
func TestSendOmitsEmptyOptionalFields(t *testing.T) {
	var seen capture
	server := serve(t, 200, `{"ok":true}`, &seen)

	client := New(envOf(map[string]string{"AGENTIC_OS_MESSENGER_URL": server.URL}))
	if _, err := client.Send(context.Background(), SendRequest{Channel: "ops", Text: "hi"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(seen.body, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"to", "reply_to", "file"} {
		if _, present := got[key]; present {
			t.Errorf("body carries an empty %q: %s", key, seen.body)
		}
	}
}

// A hub failure has to reach the user as the hub's own words. Returning only
// "500 Internal Server Error" hid the actual cause ("unknown channel") and made
// a one-line typo look like a broken server.
func TestNon2xxSurfacesTheServersMessage(t *testing.T) {
	var seen capture
	server := serve(t, http.StatusBadRequest, `{"error":"unknown channel \"opz\""}`, &seen)

	client := New(envOf(map[string]string{"AGENTIC_OS_MESSENGER_URL": server.URL}))
	_, err := client.Send(context.Background(), SendRequest{Channel: "opz", Text: "hi"})
	if err == nil {
		t.Fatal("want an error for a 400 response")
	}
	if !strings.Contains(err.Error(), `unknown channel "opz"`) {
		t.Fatalf("error %q does not carry the hub's message", err)
	}
	if !strings.Contains(err.Error(), "400") {
		t.Fatalf("error %q should still name the status", err)
	}
}

func TestNon2xxWithoutABodyStillFails(t *testing.T) {
	var seen capture
	server := serve(t, http.StatusInternalServerError, "", &seen)

	client := New(envOf(map[string]string{"AGENTIC_OS_MESSENGER_URL": server.URL}))
	if _, err := client.Health(context.Background()); err == nil {
		t.Fatal("want an error for a 500 with no JSON body")
	}
}

func TestHealthDecodesChannels(t *testing.T) {
	var seen capture
	server := serve(t, 200, `{"ok":true,"service":"messenger","channels":{"ops":"telegram"}}`, &seen)

	client := New(envOf(map[string]string{"AGENTIC_OS_MESSENGER_URL": server.URL}))
	health, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if !health.OK || health.Service != "messenger" || health.Channels["ops"] != "telegram" {
		t.Fatalf("health = %+v", health)
	}
	if seen.method != http.MethodGet || seen.path != "/health" {
		t.Fatalf("%s %s, want GET /health", seen.method, seen.path)
	}
	if len(seen.body) != 0 {
		t.Fatalf("a GET must not carry a body, got %q", seen.body)
	}
}

// The cursor is what makes polling incremental; sending it in the wrong place
// (or not at all) replays the whole inbox on every tick.
func TestInboxSendsTheCursorAsAQueryParameter(t *testing.T) {
	var seen capture
	server := serve(t, 200, `{"messages":[{"channel":"ops"}],"next":8}`, &seen)

	client := New(envOf(map[string]string{"AGENTIC_OS_MESSENGER_URL": server.URL}))
	inbox, err := client.Inbox(context.Background(), 7)
	if err != nil {
		t.Fatalf("inbox: %v", err)
	}
	if seen.path != "/inbox" || seen.query != "since=7" {
		t.Fatalf("got %s?%s, want /inbox?since=7", seen.path, seen.query)
	}
	if len(inbox.Messages) != 1 || inbox.Next != 8 {
		t.Fatalf("inbox = %+v", inbox)
	}
}

// `doctor` asks whether a hub is running at all, so an unreachable hub must
// come back as an error naming the address rather than hanging or panicking.
func TestUnreachableHubNamesTheAddress(t *testing.T) {
	// Port 1 on loopback: nothing listens, and the refusal is immediate.
	client := New(envOf(map[string]string{"AGENTIC_OS_MESSENGER_URL": "http://127.0.0.1:1"}))
	_, err := client.HealthWithin(2 * time.Second)
	if err == nil {
		t.Fatal("want an error when no hub is listening")
	}
	if !strings.Contains(err.Error(), "127.0.0.1:1") {
		t.Fatalf("error %q should name the hub address", err)
	}
}

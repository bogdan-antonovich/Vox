package hub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// chatResponse builds a minimal non-streaming OpenAI-compatible chat completion
// body whose assistant message content is the given string.
func chatResponse(content string) []byte {
	body := map[string]any{
		"id":     "chatcmpl-test",
		"object": "chat.completion",
		"model":  "test-model",
		"choices": []map[string]any{
			{
				"index":         0,
				"finish_reason": "stop",
				"message":       map[string]string{"role": "assistant", "content": content},
			},
		},
	}
	data, _ := json.Marshal(body)
	return data
}

// segResponse marshals a validatorResult into the assistant content payload.
func segResponse(complete []string, remainder string) string {
	data, _ := json.Marshal(validatorResult{Complete: complete, Remainder: remainder})
	return string(data)
}

// newSegmentationServer returns a server that replies with the supplied contents
// in order, one per request. Once exhausted it repeats the last entry.
func newSegmentationServer(t *testing.T, contents []string) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	idx := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		c := contents[len(contents)-1]
		if idx < len(contents) {
			c = contents[idx]
		}
		idx++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(chatResponse(c))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTestValidator(t *testing.T, baseURL string) (*Validator, *StringChan, *StringChan) {
	t.Helper()
	in := NewStringChanBuf(8)
	out := NewStringChanBuf(8)
	v := &Validator{
		ApiKey:    "test-key",
		Model:     "test-model",
		BaseURL:   baseURL,
		Lang:      "en",
		tokens:    in,
		validated: out,
		log:       zap.NewNop(),
	}
	return v, in, out
}

// collect drains a StringChan until it is closed and returns the values.
func collect(out *StringChan) []string {
	got := make([]string, 0, cap(out.Ch))
	for s := range out.Ch {
		got = append(got, s)
	}
	return got
}

func runValidator(t *testing.T, v *Validator, out *StringChan, feed func()) []string {
	t.Helper()
	resultCh := make(chan []string, 1)
	go func() { resultCh <- collect(out) }()

	errCh := make(chan error, 1)
	go func() {
		defer out.Close()
		errCh <- v.do(context.Background())
	}()

	feed()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("validator.do returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("validator.do did not finish in time")
	}

	select {
	case got := <-resultCh:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("collector did not finish in time")
		return nil
	}
}

func TestValidator_SplitsMultipleThoughtsInOrder(t *testing.T) {
	srv := newSegmentationServer(t, []string{
		segResponse([]string{"First thought.", "Second thought."}, "trailing piece"),
	})
	v, in, out := newTestValidator(t, srv.URL)

	got := runValidator(t, v, out, func() {
		in.Ch <- "First thought. Second thought. trailing piece"
		in.Close()
	})

	want := []string{"First thought.", "Second thought.", "trailing piece"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order mismatch at %d: got %q want %q (full %v)", i, got[i], want[i], got)
		}
	}
}

func TestValidator_BuffersIncompleteUntilComplete(t *testing.T) {
	srv := newSegmentationServer(t, []string{
		segResponse(nil, "Hello"),                 // first chunk: nothing complete yet
		segResponse([]string{"Hello world."}, ""), // after second chunk: complete
	})
	v, in, out := newTestValidator(t, srv.URL)

	got := runValidator(t, v, out, func() {
		in.Ch <- "Hello"
		in.Ch <- "world."
		in.Close()
	})

	want := []string{"Hello world."}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestValidator_FlushesRemainderOnClose(t *testing.T) {
	srv := newSegmentationServer(t, []string{
		segResponse(nil, "A dangling tail"),
	})
	v, in, out := newTestValidator(t, srv.URL)

	got := runValidator(t, v, out, func() {
		in.Ch <- "A dangling tail"
		in.Close()
	})

	if len(got) != 1 || got[0] != "A dangling tail" {
		t.Fatalf("expected dangling remainder to be flushed, got %v", got)
	}
}

func TestValidator_NonJSONResponseForwardsWholeText(t *testing.T) {
	srv := newSegmentationServer(t, []string{"this is not json at all"})
	v, in, out := newTestValidator(t, srv.URL)

	got := runValidator(t, v, out, func() {
		in.Ch <- "some translated text"
		in.Close()
	})

	if len(got) != 1 || got[0] != "some translated text" {
		t.Fatalf("expected whole text forwarded on non-JSON response, got %v", got)
	}
}

func TestValidator_IgnoresEmptyChunks(t *testing.T) {
	srv := newSegmentationServer(t, []string{
		segResponse([]string{"Real content."}, ""),
	})
	v, in, out := newTestValidator(t, srv.URL)

	got := runValidator(t, v, out, func() {
		in.Ch <- "   "
		in.Ch <- "Real content."
		in.Close()
	})

	if len(got) != 1 || got[0] != "Real content." {
		t.Fatalf("expected empty chunk ignored, got %v", got)
	}
}

func TestExtractJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", `{"a":1}`, `{"a":1}`},
		{"fenced", "```json\n{\"a\":1}\n```", `{"a":1}`},
		{"prose", `Sure! {"a":1} done`, `{"a":1}`},
		{"none", `no braces here`, `no braces here`},
		{"nested", `{"a":{"b":2}}`, `{"a":{"b":2}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractJSON(tc.in); got != tc.want {
				t.Fatalf("extractJSON(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

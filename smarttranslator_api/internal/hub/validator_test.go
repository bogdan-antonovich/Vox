package hub

import (
	"context"
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestValidator(t *testing.T) (*Validator, *StringChan, *StringChan) {
	t.Helper()
	in := NewStringChanBuf(8)
	out := NewStringChanBuf(8)
	v := &Validator{
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

func TestSplitThoughts(t *testing.T) {
	cases := []struct {
		name          string
		in            string
		wantComplete  []string
		wantRemainder string
	}{
		{"single complete", "Hello world.", []string{"Hello world."}, ""},
		{"multiple", "One. Two! Three?", []string{"One.", "Two!", "Three?"}, ""},
		{"trailing remainder", "Done. and more", []string{"Done."}, "and more"},
		{"no punctuation", "just a fragment", nil, "just a fragment"},
		{"ellipsis", "Well… maybe.", []string{"Well…", "maybe."}, ""},
		{"cjk", "你好。", []string{"你好。"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			complete, remainder := splitThoughts(tc.in)
			if !reflect.DeepEqual(complete, tc.wantComplete) {
				t.Fatalf("complete = %#v, want %#v", complete, tc.wantComplete)
			}
			if remainder != tc.wantRemainder {
				t.Fatalf("remainder = %q, want %q", remainder, tc.wantRemainder)
			}
		})
	}
}

func TestValidator_SplitsMultipleThoughtsInOrder(t *testing.T) {
	v, in, out := newTestValidator(t)

	got := runValidator(t, v, out, func() {
		in.Ch <- "First thought. Second thought. trailing piece"
		in.Close()
	})

	want := []string{"First thought.", "Second thought.", "trailing piece"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestValidator_BuffersIncompleteUntilComplete(t *testing.T) {
	v, in, out := newTestValidator(t)

	got := runValidator(t, v, out, func() {
		in.Ch <- "Hello"  // no terminal punctuation: held
		in.Ch <- "world." // completes the sentence
		in.Close()
	})

	want := []string{"Hello world."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestValidator_FlushesRemainderOnClose(t *testing.T) {
	v, in, out := newTestValidator(t)

	got := runValidator(t, v, out, func() {
		in.Ch <- "A dangling tail"
		in.Close()
	})

	if len(got) != 1 || got[0] != "A dangling tail" {
		t.Fatalf("expected dangling remainder to be flushed, got %v", got)
	}
}

func TestValidator_IgnoresEmptyChunks(t *testing.T) {
	v, in, out := newTestValidator(t)

	got := runValidator(t, v, out, func() {
		in.Ch <- "   "
		in.Ch <- "Real content."
		in.Close()
	})

	if len(got) != 1 || got[0] != "Real content." {
		t.Fatalf("expected empty chunk ignored, got %v", got)
	}
}

func TestValidator_SafetyValveFlushesLongUnpunctuatedBuffer(t *testing.T) {
	v, in, out := newTestValidator(t)

	// A run of words that never terminates a sentence must still be flushed once
	// it crosses maxBufferedWords, rather than stalling until the stream closes.
	words := make([]byte, 0, maxBufferedWords*5)
	for i := 0; i < maxBufferedWords+5; i++ {
		words = append(words, []byte("word ")...)
	}

	got := runValidator(t, v, out, func() {
		in.Ch <- string(words)
		in.Close()
	})

	if len(got) == 0 {
		t.Fatal("expected long unpunctuated buffer to be flushed by the safety valve")
	}
}

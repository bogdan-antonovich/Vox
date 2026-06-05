package hub

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type Channel interface {
	Close()
}

type StringChan struct {
	mu       *sync.Mutex
	Ch       chan string
	isClosed bool
}

type ErrorChan struct {
	mu       *sync.Mutex
	Ch       chan error
	isClosed bool
}

func (sc *StringChan) Close() {
	sc.mu.Lock()
	if !sc.isClosed {
		close(sc.Ch)
		sc.isClosed = true
	}
	sc.mu.Unlock()
}

func (ec *ErrorChan) Close() {
	ec.mu.Lock()
	if !ec.isClosed {
		close(ec.Ch)
		ec.isClosed = true
	}
	ec.mu.Unlock()
}

func NewStringChanBuf(size int) *StringChan {
	return &StringChan{
		mu: &sync.Mutex{},
		Ch: make(chan string, size),
	}
}

func NewErrorChanBuf(size int) *ErrorChan {
	return &ErrorChan{
		mu: &sync.Mutex{},
		Ch: make(chan error, size),
	}
}

type Stream interface {
	Bytes() []byte
	Close() error
	Collect() ([]byte, error)
	Err() error
	Next() bool
	Read(p []byte) (n int, err error)
}

type VoiceAgent interface {
	NewStream(ctx context.Context) (s Stream, err error)
	Handle(ctx context.Context, s Stream) error
	Do(ctx context.Context) error
}

type VoiceAgentBuilder interface {
	Get() VoiceAgent
	SetHub(hub *Hub)
	SetTokens(tokens *StringChan)
	SetLogger(log *zap.Logger)
	SetReference(audio []byte, text string)
}

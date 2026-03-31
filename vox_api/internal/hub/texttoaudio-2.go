package hub

import (
	"context"
	"errors"
	"io"

	"github.com/openai/openai-go"
	"go.uber.org/zap"
)

type OpenAIStream struct {
	reader io.ReadCloser
	chunk  []byte
	err    error
}

func (s *OpenAIStream) Next() bool {
	buf := make([]byte, 4096)
	n, err := s.reader.Read(buf)
	if n > 0 {
		s.chunk = buf[:n]
		return true
	}

	if err != nil && !errors.Is(err, io.EOF) {
		s.err = err
	}
	return false
}

func (s *OpenAIStream) Bytes() []byte              { return s.chunk }
func (s *OpenAIStream) Err() error                 { return s.err }
func (s *OpenAIStream) Close() error               { return s.reader.Close() }
func (s *OpenAIStream) Read(p []byte) (int, error) { return s.reader.Read(p) }
func (s *OpenAIStream) Collect() ([]byte, error)   { return io.ReadAll(s.reader) }

type OpenAI struct {
	client openai.Client
	hub    *Hub
	tokens *StringChan
	log    *zap.Logger
	text   string
}

func (f *OpenAI) Handle(ctx context.Context, s Stream) error {
	f.log.Debug("OpenAI.Handle", zap.Bool("ctx_is_nil", ctx == nil))
	for s.Next() {
		f.hub.Publish(s.Bytes())
	}

	return nil
}

func (f *OpenAI) NewStream(ctx context.Context) (s Stream, err error) {
	f.log.Debug("OpenAI.NewStream", zap.Bool("ctx_is_nil", ctx == nil))
	response, err := f.client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{ //nolint:bodyclose
		Model:          openai.SpeechModelTTS1,
		Voice:          openai.AudioSpeechNewParamsVoiceAlloy,
		Input:          f.text,
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
	})
	if err != nil {
		f.log.Error("Failed to stream TTS", zap.Error(err))
		return
	}

	return &OpenAIStream{reader: response.Body}, nil
}

func (f *OpenAI) Do(ctx context.Context) error {
	f.log.Debug("OpenAI.Do", zap.Bool("ctx_is_nil", ctx == nil))
	for {
		select {
		case <-ctx.Done():
			f.log.Debug("Context is canceled")
			return nil
		case f.text = <-f.tokens.Ch:

			s, err := f.NewStream(ctx)
			if err != nil {
				return err
			}

			f.log.Debug("OpenAI.Do got stream, calling Handle")
			if err := f.Handle(ctx, s); err != nil {
				_ = s.Close()
				return err
			}
			if err := s.Close(); err != nil {
				return err
			}

			if err := s.Err(); err != nil {
				f.log.Error("OpenAI stream error", zap.Error(err))
				return err
			}
		}
	}
}

type OpenAIBuilder struct {
	client openai.Client
	hub    *Hub
	tokens *StringChan
	log    *zap.Logger
}

func (b *OpenAIBuilder) SetReference(audio []byte, text string) {}
func (b *OpenAIBuilder) SetHub(hub *Hub)                        { b.hub = hub }
func (b *OpenAIBuilder) SetTokens(tokens *StringChan)           { b.tokens = tokens }
func (b *OpenAIBuilder) SetLogger(log *zap.Logger)              { b.log = log }
func (b *OpenAIBuilder) Get() VoiceAgent {
	return &OpenAI{
		client: b.client,
		hub:    b.hub,
		tokens: b.tokens,
		log:    b.log,
	}
}

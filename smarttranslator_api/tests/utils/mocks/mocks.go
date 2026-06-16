package mocks

import (
	"context"
	"errors"
	"smarttranslator/internal/auth"
	"smarttranslator/internal/hub"
	"smarttranslator/internal/user"
	"smarttranslator/internal/user/voice"

	"go.uber.org/zap"
)

type AuthDB struct {
	AddNewManualUserF   func(ctx context.Context, log *zap.Logger, u auth.UserInfo, hash []byte) error
	GetUserF            func(ctx context.Context, log *zap.Logger, providerID int, userProviderID string) (auth.UserInfo, bool, error)
	AddNewProviderUserF func(ctx context.Context, log *zap.Logger, u auth.UserInfo) error
	GetPasswordHashF    func(ctx context.Context, log *zap.Logger, login string) ([]byte, error)
	SaveRefreshTokenF   func(ctx context.Context, log *zap.Logger, login, refreshHash string) error
	GetRefreshTokenF    func(ctx context.Context, log *zap.Logger, login string) (string, error)
}

func (m *AuthDB) AddNewManualUser(ctx context.Context, log *zap.Logger, u auth.UserInfo, hash []byte) error {
	return m.AddNewManualUserF(ctx, log, u, hash)
}
func (m *AuthDB) GetUser(ctx context.Context, log *zap.Logger, providerID int, userProviderID string) (auth.UserInfo, bool, error) {
	return m.GetUserF(ctx, log, providerID, userProviderID)
}
func (m *AuthDB) AddNewProviderUser(ctx context.Context, log *zap.Logger, u auth.UserInfo) error {
	return m.AddNewProviderUserF(ctx, log, u)
}
func (m *AuthDB) GetPasswordHash(ctx context.Context, log *zap.Logger, login string) ([]byte, error) {
	return m.GetPasswordHashF(ctx, log, login)
}
func (m *AuthDB) SaveRefreshToken(ctx context.Context, log *zap.Logger, login, refreshHash string) error {
	return m.SaveRefreshTokenF(ctx, log, login, refreshHash)
}
func (m *AuthDB) GetRefreshToken(ctx context.Context, log *zap.Logger, login string) (string, error) {
	return m.GetRefreshTokenF(ctx, log, login)
}

type HubDB struct {
	GetReferenceF func(ctx context.Context, log *zap.Logger, userID, fileID string) (path, filetype, text string, err error)
}

func (m *HubDB) GetReference(ctx context.Context, log *zap.Logger, userID, fileID string) (string, string, string, error) {
	return m.GetReferenceF(ctx, log, userID, fileID)
}

type UserDB struct {
	GetUserInfoF func(ctx context.Context, log *zap.Logger, userID string) (user.UserInfo, error)
}

func (m *UserDB) GetUserInfo(ctx context.Context, log *zap.Logger, userID string) (user.UserInfo, error) {
	return m.GetUserInfoF(ctx, log, userID)
}

type VoiceDB struct {
	SaveNewVoiceReferenceF func(ctx context.Context, log *zap.Logger, userID, text, fileID, path, typeof string) error
	GetVoiceReferenceF     func(ctx context.Context, log *zap.Logger, userID string) ([5]voice.VoiceReference, int, error)
	DeleteVoiceReferenceF  func(ctx context.Context, log *zap.Logger, userID, fileID string) error
}

func (m *VoiceDB) SaveNewVoiceReference(ctx context.Context, log *zap.Logger, userID, text, fileID, path, typeof string) error {
	return m.SaveNewVoiceReferenceF(ctx, log, userID, text, fileID, path, typeof)
}

func (m *VoiceDB) GetVoiceReference(ctx context.Context, log *zap.Logger, userID string) ([5]voice.VoiceReference, int, error) {
	return m.GetVoiceReferenceF(ctx, log, userID)
}

func (m *VoiceDB) DeleteVoiceReference(ctx context.Context, log *zap.Logger, userID, fileID string) error {
	return m.DeleteVoiceReferenceF(ctx, log, userID, fileID)
}

// type MockFishBuilder struct {
// 	tokens *hub.StringChan
// 	hub    *hub.Hub
// 	DoFunc func(ctx context.Context) error
// }

// func (m *MockFishBuilder) SetReference(audio []byte, text string) {}
// func (m *MockFishBuilder) SetHub(h *hub.Hub) {
// 	m.hub = h
// }

// func (m *MockFishBuilder) SetTokens(tokens *hub.StringChan) {
// 	m.tokens = tokens
// }
// func (m *MockFishBuilder) SetLogger(log *zap.Logger) {}
// func (m *MockFishBuilder) Get() hub.FishAudio {
// 	return &MockFishAudio{tokens: m.tokens, hub: m.hub, DoFunc: m.DoFunc}
// }

// type MockFishAudio struct {
// 	tokens *hub.StringChan
// 	hub    *hub.Hub
// 	DoFunc func(ctx context.Context) error
// }

// func (m *MockFishAudio) StreamWebSocket(ctx context.Context, textChan <-chan string, params *fishaudio.StreamParams, opts *fishaudio.WebSocketOptions) (hub.FishStream, error) {
// 	return nil, nil
// }
// func (m *MockFishAudio) HandleStream(stream hub.FishStream) {}
// func (m *MockFishAudio) Do(ctx context.Context) error {
// 	for range m.tokens.Ch {
// 	}
// 	if m.hub != nil {
// 		m.hub.Publish([]byte("fish-audio-bytes"))
// 	}
// 	return m.DoFunc(ctx)
// }

// func HappyFishBuilder() *MockFishBuilder {
// 	return &MockFishBuilder{
// 		DoFunc: func(ctx context.Context) error { return nil },
// 	}
// }
//

type MockVoiceAgent struct {
	tokens *hub.StringChan
	hub    *hub.Hub
	DoFunc func(ctx context.Context) error
}

func (m *MockVoiceAgent) NewStream(ctx context.Context) (hub.Stream, error)   { return nil, nil }
func (m *MockVoiceAgent) Handle(ctx context.Context, stream hub.Stream) error { return nil }
func (m *MockVoiceAgent) Do(ctx context.Context) error {
	if m.DoFunc != nil {
		if err := m.DoFunc(ctx); err != nil {
			return err
		}
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-m.tokens.Ch:
			if !ok {
				if m.hub != nil {
					m.hub.Publish([]byte("fish-audio-bytes"))
				}
				return nil
			}
		}
	}
}

type MockVoiceAgentBuilder struct {
	tokens *hub.StringChan
	hub    *hub.Hub
	DoFunc func(ctx context.Context) error
}

func (m *MockVoiceAgentBuilder) SetReference(audio []byte, text string) {}
func (m *MockVoiceAgentBuilder) SetHub(h *hub.Hub) {
	m.hub = h
}

func (m *MockVoiceAgentBuilder) SetTokens(tokens *hub.StringChan) {
	m.tokens = tokens
}
func (m *MockVoiceAgentBuilder) SetLogger(log *zap.Logger) {}
func (m *MockVoiceAgentBuilder) Get() hub.VoiceAgent {
	return &MockVoiceAgent{tokens: m.tokens, hub: m.hub, DoFunc: m.DoFunc}
}

func HappyVoiceAgentBuilder() *MockVoiceAgentBuilder {
	return &MockVoiceAgentBuilder{
		DoFunc: func(ctx context.Context) error { return nil },
	}
}

// EchoVoiceAgent publishes every token it receives (the validated complete
// thoughts) to the hub as raw bytes, so tests can assert exactly what the
// validator forwarded downstream.
type EchoVoiceAgent struct {
	tokens *hub.StringChan
	hub    *hub.Hub
}

func (m *EchoVoiceAgent) NewStream(ctx context.Context) (hub.Stream, error)   { return nil, nil }
func (m *EchoVoiceAgent) Handle(ctx context.Context, stream hub.Stream) error { return nil }
func (m *EchoVoiceAgent) Do(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case tok, ok := <-m.tokens.Ch:
			if !ok {
				return nil
			}
			if m.hub != nil {
				m.hub.Publish([]byte(tok))
			}
		}
	}
}

type EchoVoiceAgentBuilder struct {
	tokens *hub.StringChan
	hub    *hub.Hub
}

func (m *EchoVoiceAgentBuilder) SetReference(audio []byte, text string) {}
func (m *EchoVoiceAgentBuilder) SetHub(h *hub.Hub)                      { m.hub = h }
func (m *EchoVoiceAgentBuilder) SetTokens(tokens *hub.StringChan)       { m.tokens = tokens }
func (m *EchoVoiceAgentBuilder) SetLogger(log *zap.Logger)              {}
func (m *EchoVoiceAgentBuilder) Get() hub.VoiceAgent {
	return &EchoVoiceAgent{tokens: m.tokens, hub: m.hub}
}

func NewEchoVoiceAgentBuilder() *EchoVoiceAgentBuilder {
	return &EchoVoiceAgentBuilder{}
}

func ErrorVoiceAgentBuilder() *MockVoiceAgentBuilder {
	return &MockVoiceAgentBuilder{
		DoFunc: func(ctx context.Context) error {
			return errors.New("voice agent error")
		},
	}
}

// func ErrorFishBuilder() *MockFishBuilder {
// 	return &MockFishBuilder{
// 		DoFunc: func(ctx context.Context) error {
// 			return errors.New("fish error")
// 		},
// 	}
// }

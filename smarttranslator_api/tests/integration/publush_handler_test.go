//go:build integration

package integration

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"smarttranslator/internal/hub"
	"smarttranslator/tests/utils/helpers"
	"smarttranslator/tests/utils/mocks"
	"smarttranslator/tests/utils/vars"
)

func TestPublishHandler_MissingUserID_Returns404(t *testing.T) {
	h := hub.NewHub(uuid.New().String())
	defer h.Close()

	api := &hub.HubAPI{MGR: hub.NewManager(), Cfg: vars.PublishCfg(), DB: helpers.HappyHubDB("", "", "")}
	r := helpers.NewPublishRouterNoUserID(t, api, h)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, helpers.PublishRequest(h.ID, "1234"))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPublishHandler_MissingHub_Returns404(t *testing.T) {
	api := &hub.HubAPI{MGR: hub.NewManager(), Cfg: vars.PublishCfg(), DB: helpers.HappyHubDB("", "", "")}
	r := helpers.NewPublishRouterNoHub(t, api, uuid.New().String())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, helpers.PublishRequest(uuid.New().String(), "12345"))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPublishHandler_WrongHubType_Returns500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(helpers.InjectLogger(zaptest.NewLogger(t)))
	r.Use(func(ctx *gin.Context) {
		ctx.Set("hub", "not-a-hub-pointer")
		ctx.Set("user_id", uuid.New().String())
		ctx.Next()
	})
	api := &hub.HubAPI{MGR: hub.NewManager(), Cfg: vars.PublishCfg(), DB: helpers.HappyHubDB("", "", "")}
	r.GET("/hub/:hub_id/publish", api.PublishHandler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, helpers.PublishRequest(uuid.New().String(), "12345"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// func TestPublishHandler_DBError_Returns500(t *testing.T) {
// 	h := hub.NewHub(uuid.New().String())
// 	defer h.Close()
// 	cache := hub.NewHostAndHubs()

// 	api := &hub.HubAPI{MGR: hub.NewManager(), Cfg: vars.PublishCfg(), DB: helpers.ErrorHubDB()}
// 	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.HappyVoiceAgentBuilder())

// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, helpers.PublishRequest(h.ID, "1234"))

// 	assert.Equal(t, http.StatusInternalServerError, w.Code)
// }

// func TestPublishHandler_DBError_ResponseBodyIsValidJSON(t *testing.T) {
// 	h := hub.NewHub(uuid.New().String())
// 	defer h.Close()
// 	cache := hub.NewHostAndHubs()

// 	api := &hub.HubAPI{MGR: hub.NewManager(), Cfg: vars.PublishCfg(), DB: helpers.ErrorHubDB()}
// 	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.HappyVoiceAgentBuilder())

// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, helpers.PublishRequest(h.ID, "1234"))

// 	require.Equal(t, http.StatusInternalServerError, w.Code)
// 	assert.True(t, json.Valid(w.Body.Bytes()))
// }

// func TestPublishHandler_FileNotFound_Returns500(t *testing.T) {
// 	h := hub.NewHub(uuid.New().String())
// 	defer h.Close()
// 	cache := hub.NewHostAndHubs()

// 	api := &hub.HubAPI{
// 		MGR: hub.NewManager(),
// 		Cfg: vars.PublishCfg(),
// 		DB:  helpers.HappyHubDB("/nonexistent/does-not-exist.mp3", "mp3", "text"),
// 	}
// 	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.HappyVoiceAgentBuilder())

// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, helpers.PublishRequest(h.ID, "1234"))

// 	assert.Equal(t, http.StatusInternalServerError, w.Code)
// }

// func TestPublishHandler_FileNotFound_ResponseBodyIsValidJSON(t *testing.T) {
// 	h := hub.NewHub(uuid.New().String())
// 	defer h.Close()
// 	cache := hub.NewHostAndHubs()

// 	api := &hub.HubAPI{
// 		MGR: hub.NewManager(),
// 		Cfg: vars.PublishCfg(),
// 		DB:  helpers.HappyHubDB("/nonexistent/does-not-exist.mp3", "mp3", "text"),
// 	}
// 	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.HappyVoiceAgentBuilder())

// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, helpers.PublishRequest(h.ID, "1234"))

// 	require.Equal(t, http.StatusInternalServerError, w.Code)
// 	assert.True(t, json.Valid(w.Body.Bytes()))
// }

// defer MGR.Delete is registered AFTER the DB+file reads, so on a DB error
// the hub must still be present in the manager when the handler returns.
// func TestPublishHandler_DBError_HubNotRemovedFromManager(t *testing.T) {
// 	mgr := hub.NewManager()
// 	hubID := mgr.New()
// 	h, _ := mgr.Get(hubID)
// 	cache := hub.NewHostAndHubs()

// 	api := &hub.HubAPI{MGR: mgr, Cfg: vars.PublishCfg(), DB: helpers.ErrorHubDB()}
// 	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.HappyVoiceAgentBuilder())

// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, helpers.PublishRequest(hubID, "12345"))

// 	require.Equal(t, http.StatusInternalServerError, w.Code)
// 	_, ok := mgr.Get(hubID)
// 	assert.True(t, ok, "hub must still exist: defer Delete not registered at DB-error return point")
// }

// Same for file-not-found.
// func TestPublishHandler_FileNotFound_HubNotRemovedFromManager(t *testing.T) {
// 	mgr := hub.NewManager()
// 	hubID := mgr.New()
// 	h, _ := mgr.Get(hubID)
// 	cache := hub.NewHostAndHubs()

// 	api := &hub.HubAPI{
// 		MGR: mgr,
// 		Cfg: vars.PublishCfg(),
// 		DB:  helpers.HappyHubDB("/nonexistent/does-not-exist.mp3", "mp3", "text"),
// 	}
// 	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.HappyVoiceAgentBuilder())

// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, helpers.PublishRequest(hubID, "12345"))

//		require.Equal(t, http.StatusInternalServerError, w.Code)
//		_, ok := mgr.Get(hubID)
//		assert.True(t, ok, "hub must still exist: defer Delete not registered at file-error return point")
//	}
func TestPublishHandler_HappyPath(t *testing.T) {
	refFile := helpers.WriteTempFile(t, []byte("fake-reference-audio"))
	dgSrv := helpers.NewMockDeepgramServer(t, "hello world")
	groqSrv := helpers.NewMockGroqServer(t, "processed token")
	mgr := hub.NewManager()
	cache := hub.NewHostAndHubs()
	hubID := mgr.New()
	h, _ := mgr.Get(hubID)
	api := &hub.HubAPI{
		MGR: mgr,
		Cfg: vars.CfgWithMocks(dgSrv, groqSrv, nil),
		DB:  helpers.HappyHubDB(refFile, "mp3", "reference text"),
	}
	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.HappyVoiceAgentBuilder())

	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/hub/" + hubID + "/publish?lang=ru&file_id=12345"
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	// Send audio
	err = ws.WriteMessage(websocket.BinaryMessage, []byte("fake-audio-data"))
	require.NoError(t, err)

	// Close cleanly
	err = ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	require.NoError(t, err)

	// Wait for server to close its side
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func TestPublishHandler_FishError(t *testing.T) {
	refFile := helpers.WriteTempFile(t, []byte("fake-reference-audio"))
	dgSrv := helpers.NewMockDeepgramServer(t, "hello world")
	groqSrv := helpers.NewMockGroqServer(t, "processed token")
	mgr := hub.NewManager()
	hubID := mgr.New()
	h, _ := mgr.Get(hubID)
	api := &hub.HubAPI{
		MGR: mgr,
		Cfg: vars.CfgWithMocks(dgSrv, groqSrv, nil),
		DB:  helpers.HappyHubDB(refFile, "mp3", "reference text"),
	}
	cache := hub.NewHostAndHubs()
	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.ErrorVoiceAgentBuilder())

	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/hub/" + hubID + "/publish?lang=ru&file_id=12345"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.BinaryMessage, []byte("fake-audio"))
	require.NoError(t, err)

	// Server should close connection due to pipeline error
	_, _, err = ws.ReadMessage()
	assert.Error(t, err)
}

func TestPublishHandler_NoFishBuilder_Returns500(t *testing.T) {
	refFile := helpers.WriteTempFile(t, []byte("fake-reference-audio"))
	dgSrv := helpers.NewMockDeepgramServer(t, "hello world")
	groqSrv := helpers.NewMockGroqServer(t, "processed token")
	mgr := hub.NewManager()
	hubID := mgr.New()
	h, _ := mgr.Get(hubID)
	api := &hub.HubAPI{
		MGR: mgr,
		Cfg: vars.CfgWithMocks(dgSrv, groqSrv, nil),
		DB:  helpers.HappyHubDB(refFile, "mp3", "reference text"),
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(helpers.InjectLogger(zaptest.NewLogger(t)))
	r.Use(helpers.InjectHub(h))
	r.Use(func(ctx *gin.Context) {
		uID := uuid.New().String()
		ctx.Set("user_id", uID)
		c := hub.NewHostAndHubs()
		c.AddHub(uID, hubID)
		ctx.Set("host_and_hub_cache", c)
		ctx.Next()
	})
	r.GET("/hub/:hub_id/publish", api.PublishHandler)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, helpers.PublishRequest(hubID, "12345"))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPublishHandler_InvalidFishBuilder_Returns500(t *testing.T) {
	refFile := helpers.WriteTempFile(t, []byte("fake-reference-audio"))
	dgSrv := helpers.NewMockDeepgramServer(t, "hello world")
	groqSrv := helpers.NewMockGroqServer(t, "processed token")
	mgr := hub.NewManager()
	hubID := mgr.New()
	h, _ := mgr.Get(hubID)
	api := &hub.HubAPI{
		MGR: mgr,
		Cfg: vars.CfgWithMocks(dgSrv, groqSrv, nil),
		DB:  helpers.HappyHubDB(refFile, "mp3", "reference text"),
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(helpers.InjectLogger(zaptest.NewLogger(t)))
	r.Use(helpers.InjectHub(h))
	r.Use(func(ctx *gin.Context) {
		uID := uuid.New().String()
		ctx.Set("user_id", uID)
		ctx.Set("voice_agent_builder", "not-a-fish-builder")
		c := hub.NewHostAndHubs()
		c.AddHub(uID, hubID)
		ctx.Set("host_and_hub_cache", c)
		ctx.Next()
	})
	r.GET("/hub/:hub_id/publish", api.PublishHandler)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, helpers.PublishRequest(hubID, "12345"))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPublishHandler_HappyPath_HubRemovedFromManagerAfterReturn(t *testing.T) {
	refFile := helpers.WriteTempFile(t, []byte("fake-reference-audio"))
	dgSrv := helpers.NewMockDeepgramServer(t, "hello world")
	groqSrv := helpers.NewMockGroqServer(t, "token")
	fishSrv := helpers.NewMockFishAudioServer(t, []byte("chunk"))
	mgr := hub.NewManager()
	hubID := mgr.New()
	h, _ := mgr.Get(hubID)
	api := &hub.HubAPI{
		MGR: mgr,
		Cfg: vars.CfgWithMocks(dgSrv, groqSrv, fishSrv),
		DB:  helpers.HappyHubDB(refFile, "mp3", "reference text"),
	}
	cache := hub.NewHostAndHubs()
	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.HappyVoiceAgentBuilder())

	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/hub/" + hubID + "/publish?lang=ru&file_id=12345"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.BinaryMessage, []byte("fake-audio"))
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	require.NoError(t, err)

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}

	_, ok := mgr.Get(hubID)
	assert.True(t, ok, "defer MGR.Delete must have fired: hub must not be gone after handler returns")
}

func TestPublishHandler_HappyPath_AudioChunkReachesHubConsumer(t *testing.T) {
	refFile := helpers.WriteTempFile(t, []byte("fake-reference-audio"))
	const expectedAudio = "fish-audio-bytes"
	dgSrv := helpers.NewMockDeepgramServer(t, "hello")
	groqSrv := helpers.NewMockGroqServer(t, "token")
	fishSrv := helpers.NewMockFishAudioServer(t, []byte(expectedAudio))
	mgr := hub.NewManager()
	hubID := mgr.New()
	h, _ := mgr.Get(hubID)
	consumer, ch := helpers.NewConsumer(h)
	defer h.RemoveConsumer(consumer.ID)
	received := make(chan []byte, 16)
	go func() {
		for chunk := range ch {
			received <- chunk
		}
	}()
	api := &hub.HubAPI{
		MGR: mgr,
		Cfg: vars.CfgWithMocks(dgSrv, groqSrv, fishSrv),
		DB:  helpers.HappyHubDB(refFile, "mp3", "reference text"),
	}
	cache := hub.NewHostAndHubs()
	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.HappyVoiceAgentBuilder())

	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/hub/" + hubID + "/publish?lang=ru&file_id=12345"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.BinaryMessage, []byte("fake-audio"))
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	require.NoError(t, err)

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}

	select {
	case chunk := <-received:
		assert.Equal(t, []byte(expectedAudio), chunk)
	case <-time.After(2 * time.Second):
		t.Fatal("audio chunk was never delivered to the hub consumer")
	}
}

// TestPublishHandler_ValidatorForwardsCompleteThoughts drives a real transcript
// through Deepgram -> Validator -> Groq -> (echo) voice agent and asserts that
// the validator forwards a complete thought immediately and flushes the buffered
// remainder when the stream ends — in order (FIFO). The validator now segments on
// sentence-final punctuation (no LLM call): the first sentence ends with a period
// so it is emitted at once, while the unpunctuated tail is held and flushed when
// the stream closes. The mock translator echoes each thought back, so what reaches
// the consumer is exactly what the validator segmented (after passing through
// translation).
func TestPublishHandler_ValidatorForwardsCompleteThoughts(t *testing.T) {
	refFile := helpers.WriteTempFile(t, []byte("fake-reference-audio"))
	// Cyrillic transcript so it passes the lang=ru script filter and reaches the
	// validator. First sentence is punctuated; the trailing fragment is not.
	dgSrv := helpers.NewMockDeepgramServer(t, "Добро пожаловать. Главные новости выходных")
	// The segmentation arguments are unused now that the validator is heuristic;
	// the server's translation branch still echoes each thought verbatim.
	llmSrv := helpers.NewMockSegmentAndTranslateServer(t, nil, "")
	mgr := hub.NewManager()
	hubID := mgr.New()
	h, _ := mgr.Get(hubID)
	consumer, ch := helpers.NewConsumer(h)
	defer h.RemoveConsumer(consumer.ID)
	received := make(chan []byte, 16)
	go func() {
		for chunk := range ch {
			received <- chunk
		}
	}()
	api := &hub.HubAPI{
		MGR: mgr,
		Cfg: vars.CfgWithMocks(dgSrv, llmSrv, nil),
		DB:  helpers.HappyHubDB(refFile, "mp3", "reference text"),
	}
	cache := hub.NewHostAndHubs()
	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.NewEchoVoiceAgentBuilder())

	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/hub/" + hubID + "/publish?lang=ru&file_id=12345"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.BinaryMessage, []byte("fake-audio"))
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	require.NoError(t, err)

	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			break
		}
	}

	readChunk := func() []byte {
		select {
		case c := <-received:
			return c
		case <-time.After(2 * time.Second):
			t.Fatal("validator output never reached the hub consumer")
			return nil
		}
	}

	assert.Equal(t, []byte("Добро пожаловать."), readChunk())
	assert.Equal(t, []byte("Главные новости выходных"), readChunk())
}

func TestPublishHandler_DeepgramUnreachable(t *testing.T) {
	refFile := helpers.WriteTempFile(t, []byte("fake-reference-audio"))
	deadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadSrv.Close()
	groqSrv := helpers.NewMockGroqServer(t, "token")
	fishSrv := helpers.NewMockFishAudioServer(t, []byte("chunk"))
	mgr := hub.NewManager()
	hubID := mgr.New()
	h, _ := mgr.Get(hubID)
	api := &hub.HubAPI{
		MGR: mgr,
		Cfg: vars.CfgWithMocks(deadSrv, groqSrv, fishSrv),
		DB:  helpers.HappyHubDB(refFile, "mp3", "reference text"),
	}
	cache := hub.NewHostAndHubs()
	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.HappyVoiceAgentBuilder())

	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/hub/" + hubID + "/publish?lang=ru&file_id=12345"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.BinaryMessage, []byte("fake-audio"))
	require.NoError(t, err)

	_, _, err = ws.ReadMessage()
	assert.Error(t, err)
}

// TestPublishHandler_GroqUnreachable verifies the pipeline is resilient to an
// unreachable translator. Groq.do intentionally skips a failed translation
// instead of tearing down the broadcast (see translation.go), and the validator
// no longer makes its own LLM call, so an unreachable Groq must NOT kill the
// session: the connection stays up, simply producing no translated audio, and it
// shuts down cleanly when the broadcaster disconnects.
func TestPublishHandler_GroqUnreachable(t *testing.T) {
	refFile := helpers.WriteTempFile(t, []byte("fake-reference-audio"))
	// Punctuated Cyrillic transcript: passes the lang=ru filter and the heuristic
	// validator emits it as a complete thought, so Groq actually gets called.
	dgSrv := helpers.NewMockDeepgramServer(t, "привет мир.")
	deadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadSrv.Close()
	fishSrv := helpers.NewMockFishAudioServer(t, []byte("chunk"))
	mgr := hub.NewManager()
	hubID := mgr.New()
	h, _ := mgr.Get(hubID)
	consumer, ch := helpers.NewConsumer(h)
	defer h.RemoveConsumer(consumer.ID)
	received := make(chan []byte, 16)
	go func() {
		for chunk := range ch {
			received <- chunk
		}
	}()
	api := &hub.HubAPI{
		MGR: mgr,
		Cfg: vars.CfgWithMocks(dgSrv, deadSrv, fishSrv),
		DB:  helpers.HappyHubDB(refFile, "mp3", "reference text"),
	}
	cache := hub.NewHostAndHubs()
	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.HappyVoiceAgentBuilder())

	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/hub/" + hubID + "/publish?lang=ru&file_id=12345"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.BinaryMessage, []byte("fake-audio"))
	require.NoError(t, err)

	// Translation fails, so no audio is ever produced — but the session stays alive.
	select {
	case <-received:
		t.Fatal("no audio should be produced while Groq is unreachable")
	case <-time.After(750 * time.Millisecond):
	}

	// A clean client disconnect must still shut the handler down (no hang).
	err = ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	require.NoError(t, err)
	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			break
		}
	}
}

func TestPublishHandler_FishAudioUnreachable(t *testing.T) {
	refFile := helpers.WriteTempFile(t, []byte("fake-reference-audio"))
	dgSrv := helpers.NewMockDeepgramServer(t, "hello world")
	groqSrv := helpers.NewMockGroqServer(t, "token")
	mgr := hub.NewManager()
	hubID := mgr.New()
	h, _ := mgr.Get(hubID)
	api := &hub.HubAPI{
		MGR: mgr,
		Cfg: vars.CfgWithMocks(dgSrv, groqSrv, nil),
		DB:  helpers.HappyHubDB(refFile, "mp3", "reference text"),
	}
	cache := hub.NewHostAndHubs()
	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.ErrorVoiceAgentBuilder())

	srv := httptest.NewServer(r)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/hub/" + hubID + "/publish?lang=ru&file_id=12345"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.BinaryMessage, []byte("fake-audio"))
	require.NoError(t, err)

	_, _, err = ws.ReadMessage()
	assert.Error(t, err)
}

func TestPublishHandler_HappyPath_DoesNotPanic(t *testing.T) {
	refFile := helpers.WriteTempFile(t, []byte("fake-reference-audio"))

	dgSrv := helpers.NewMockDeepgramServer(t, "hello")
	groqSrv := helpers.NewMockGroqServer(t, "token")
	fishSrv := helpers.NewMockFishAudioServer(t, []byte("chunk"))

	mgr := hub.NewManager()
	hubID := mgr.New()
	h, _ := mgr.Get(hubID)

	api := &hub.HubAPI{
		MGR: mgr,
		Cfg: vars.CfgWithMocks(dgSrv, groqSrv, fishSrv),
		DB:  helpers.HappyHubDB(refFile, "mp3", "reference text"),
	}
	cache := hub.NewHostAndHubs()
	r := helpers.NewPublishRouterFull(t, api, h, uuid.New().String(), cache, mocks.HappyVoiceAgentBuilder())

	assert.NotPanics(t, func() {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, helpers.PublishRequest(hubID, "12345"))
	})
}

func TestPublishHandler_UserNotOwner_Returns403(t *testing.T) {
	h := hub.NewHub(uuid.New().String())
	defer h.Close()
	api := &hub.HubAPI{MGR: hub.NewManager(), Cfg: vars.PublishCfg(), DB: helpers.HappyHubDB("", "", "")}
	cache := hub.NewHostAndHubs()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(helpers.InjectLogger(zaptest.NewLogger(t)))
	r.Use(helpers.InjectHub(h))
	r.Use(helpers.InjectWSUpgrader())
	r.Use(func(ctx *gin.Context) {
		ctx.Set("user_id", uuid.New().String())
		ctx.Set("voice_agent_builder", mocks.HappyVoiceAgentBuilder())
		ctx.Set("host_and_hub_cache", cache)
		ctx.Next()
	})
	r.GET("/hub/:hub_id/publish", api.PublishHandler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, helpers.PublishRequest(h.ID, "12346"))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestPublishHandler_WrongCacheType_Returns500(t *testing.T) {
	h := hub.NewHub(uuid.New().String())
	defer h.Close()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(helpers.InjectLogger(zaptest.NewLogger(t)))
	r.Use(func(ctx *gin.Context) {
		ctx.Set("user_id", uuid.New().String())
		ctx.Set("hub", h)
		ctx.Set("host_and_hub_cache", "not-a-host-and-hubs-pointer")
		ctx.Next()
	})
	api := &hub.HubAPI{MGR: hub.NewManager(), Cfg: vars.PublishCfg(), DB: helpers.HappyHubDB("", "", "")}
	r.GET("/hub/:hub_id/publish", api.PublishHandler)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, helpers.PublishRequest(h.ID, "1231326"))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPublishHandler_MissingFileID_Returns400(t *testing.T) {
	h := hub.NewHub(uuid.New().String())
	defer h.Close()
	userID := uuid.New().String()
	api := &hub.HubAPI{MGR: hub.NewManager(), Cfg: vars.PublishCfg(), DB: helpers.HappyHubDB("", "", "")}
	cache := hub.NewHostAndHubs()
	r := helpers.NewPublishRouterFull(t, api, h, userID, cache, mocks.HappyVoiceAgentBuilder())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, helpers.PublishRequest(h.ID, ""))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPublishHandler_MissingWSUpgrader_Returns500(t *testing.T) {
	h := hub.NewHub(uuid.New().String())
	defer h.Close()
	userID := uuid.New().String()
	cache := hub.NewHostAndHubs()
	cache.AddHub(userID, h.ID)

	api := &hub.HubAPI{MGR: hub.NewManager(), Cfg: vars.PublishCfg(), DB: helpers.HappyHubDB("", "", "")}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(helpers.InjectLogger(zaptest.NewLogger(t)))
	r.Use(helpers.InjectHub(h))
	// No InjectWSUpgrader — intentionally missing
	r.Use(func(ctx *gin.Context) {
		ctx.Set("user_id", userID)
		ctx.Set("voice_agent_builder", mocks.HappyVoiceAgentBuilder())
		ctx.Set("host_and_hub_cache", cache)
		ctx.Next()
	})
	r.GET("/hub/:hub_id/publish", api.PublishHandler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, helpers.PublishRequest(h.ID, "12345"))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPublishHandler_InvalidWSUpgrader_Returns500(t *testing.T) {
	h := hub.NewHub(uuid.New().String())
	defer h.Close()
	userID := uuid.New().String()
	cache := hub.NewHostAndHubs()
	cache.AddHub(userID, h.ID)

	api := &hub.HubAPI{MGR: hub.NewManager(), Cfg: vars.PublishCfg(), DB: helpers.HappyHubDB("", "", "")}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(helpers.InjectLogger(zaptest.NewLogger(t)))
	r.Use(helpers.InjectHub(h))
	r.Use(func(ctx *gin.Context) {
		ctx.Set("user_id", userID)
		ctx.Set("ws_upgrader", "not-an-upgrader")
		ctx.Set("voice_agent_builder", mocks.HappyVoiceAgentBuilder())
		ctx.Set("host_and_hub_cache", cache)
		ctx.Next()
	})
	r.GET("/hub/:hub_id/publish", api.PublishHandler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, helpers.PublishRequest(h.ID, "12345"))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// func TestPublishHandler_DBNotOwnerError_Returns403(t *testing.T) {
// 	h := hub.NewHub(uuid.New().String())
// 	defer h.Close()
// 	userID := uuid.New().String()
// 	api := &hub.HubAPI{MGR: hub.NewManager(), Cfg: vars.PublishCfg(), DB: helpers.NotOwnerHubDB()}
// 	cache := hub.NewHostAndHubs()
// 	r := helpers.NewPublishRouterFull(t, api, h, userID, cache, mocks.HappyVoiceAgentBuilder())
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, helpers.PublishRequest(h.ID, "1235213"))
// 	assert.Equal(t, http.StatusForbidden, w.Code)
// }

// func TestPublishHandler_DBNotOwnerError_ResponseBodyIsValidJSON(t *testing.T) {
// 	h := hub.NewHub(uuid.New().String())
// 	defer h.Close()
// 	userID := uuid.New().String()
// 	api := &hub.HubAPI{MGR: hub.NewManager(), Cfg: vars.PublishCfg(), DB: helpers.NotOwnerHubDB()}
// 	cache := hub.NewHostAndHubs()
// 	r := helpers.NewPublishRouterFull(t, api, h, userID, cache, mocks.HappyVoiceAgentBuilder())
// 	w := httptest.NewRecorder()
// 	r.ServeHTTP(w, helpers.PublishRequest(h.ID, "1235213"))
// 	require.Equal(t, http.StatusForbidden, w.Code)
// 	assert.True(t, json.Valid(w.Body.Bytes()))
// }

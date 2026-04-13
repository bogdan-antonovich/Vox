package hub

import (
	"io"
	"net/http"
	"strings"
	"vox/pkg/helpers"
	mod "vox/pkg/models"

	interfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces"
	fishaudio "github.com/fishaudio/fish-audio-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type HubAPI struct {
	DB  HubDB
	Cfg *mod.Config
	MGR *Manager
}

func isValHub(ctx *gin.Context, key string) (hub *Hub, ok bool) {
	val, _ok := ctx.Get(key)
	if !_ok {
		return hub, ok
	}

	switch v := val.(type) {
	case *Hub:
		hub = v
		ok = true
	default:
		return hub, ok
	}
	return hub, ok
}

func (h *HubAPI) PutCache(hostAndHubs *HostAndHubs) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set("host_and_hub_cache", hostAndHubs)
		ctx.Next()
	}
}

func (h *HubAPI) IsContentTypeValid(ctx *gin.Context) {
	log := mod.GetLogger(ctx)
	contentType := ctx.GetHeader("Content-Type")
	if !strings.HasPrefix(contentType, "audio/") && contentType != "application/octet-stream" {
		ctx.Abort()
		log.Warn("Invalid content-type", zap.String("content_type", contentType))
		ctx.Data(http.StatusBadRequest, mod.APP_JSON, mod.HttpError(mod.INVALID_CONTENT_TYPE_CODE, mod.INVALID_CONTENT_TYPE_MSG))
		return
	}
	log.Debug("Content-type is valid", zap.String("content_type", contentType))
	ctx.Next()
}

func (h *HubAPI) IsHubIDValid(ctx *gin.Context) {
	log := mod.GetLogger(ctx)
	hubID := ctx.Param("hub_id")
	if hubID == "" {
		ctx.Abort()
		log.Error("Invalid hub id", zap.Bool("hub_id_is_empty", hubID == ""))
		ctx.Data(http.StatusNotFound, mod.APP_JSON, mod.HttpError(mod.INVALID_URL_CODE, mod.INVALID_URL_MSG))
		return
	}

	hub, ok := h.MGR.Get(hubID)
	if !ok {
		ctx.Abort()
		log.Warn("Invalid hub id", zap.String("hub_id", hubID))
		ctx.Data(http.StatusNotFound, mod.APP_JSON, mod.HttpError(mod.INVALID_URL_CODE, mod.INVALID_URL_MSG))
		return
	}
	ctx.Set("hub", hub)
	log.Debug("Hub id is valid", zap.String("hub_id", hubID))
	ctx.Next()
}

// NewHubHandler godoc
// @Summary      Create a new hub
// @Description  Creates a new hub for the given user and returns the generated hub ID.
// @Tags         hub
// @Produce      json
// @Success      201   {object}  map[string]string  "Hub created successfully"
// @Failure      400   {object}  models.HttpErrorResponse  "Request body is unreadable or invalid JSON"
// @Failure      403   {object}  models.HttpErrorResponse  "Unauthorized"
// @Failure      500   {object}  models.HttpErrorResponse  "Internal server error"
// @Security     CookieAuth
// @Router       /hub [post]
func (h *HubAPI) NewHubHandler(ctx *gin.Context) {
	log := mod.GetLogger(ctx)
	val, ok := ctx.Get("host_and_hub_cache")
	if !ok {
		log.Error("Invalid host_and_hub_cache type")
		ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
		return
	}

	userID, ok := helpers.IsValString(ctx, "user_id")
	if !ok {
		log.Error("user_id is missing")
		ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
		return
	}

	switch cache := val.(type) {
	case *HostAndHubs:
		hubID := h.MGR.New()
		log.Debug("New hub created", zap.String("hub_id", hubID))
		cache.AddHub(userID, hubID)
		ctx.Data(http.StatusCreated, mod.APP_JSON, []byte(`{"hub_id": "`+hubID+`"}`))
	default:
		log.Error("Invalid host_and_hub_cache type")
		ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
	}
}

// ReconnectHandler redirects user to hub publish page
// @Summary      Reconnect to hub
// @Description  Checks if the user is the owner of the specified hub and redirects to the frontend publish page
// @Tags         hub
// @Param        hub_id  path  string        true  "Hub ID"
// @Success      307  "Temporary redirect to frontend hub publish page"
// @Failure      400  {object}  models.HttpErrorResponse  "Invalid request body"
// @Failure      403  {object}  models.HttpErrorResponse  "User is not authenticated or not the hub owner"
// @Failure      500  {object}  models.HttpErrorResponse  "Internal server error"
// @Security     CookieAuth
// @Router       /hub/{hub_id}/reconnect [get]
func (h *HubAPI) ReconnectHandler(ctx *gin.Context) {
	log := mod.GetLogger(ctx)

	val, ok := ctx.Get("host_and_hub_cache")
	if !ok {
		log.Error("Invalid host_and_hub_cache type")
		ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
		return
	}

	userID, ok := helpers.IsValString(ctx, "user_id")
	if !ok {
		log.Error("user_id is missing")
		ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
		return
	}

	hubID := ctx.Param("hub_id")
	isOwner := false

	switch cache := val.(type) {
	case *HostAndHubs:
		for _, id := range cache.GetHubs(userID) {
			if id == hubID {
				isOwner = true
				break
			}
		}
		if !isOwner {
			ctx.Data(http.StatusForbidden, mod.APP_JSON, mod.HttpError(mod.FORBIDDEN_CODE, mod.FORBIDDEN_MSG))
			return
		}

		ctx.Redirect(http.StatusTemporaryRedirect, h.Cfg.FrontendURL+"/hub/"+hubID+"/publish")
	default:
		log.Error("Invalid host_and_hub_cache type")
		ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
	}
}

// DeleteHubHandler deletes a hub by ID
// @Summary      Delete hub
// @Description  Deletes the specified hub if the authenticated user is its owner
// @Tags         hub
// @Accept       json
// @Param        hub_id  path  string             true  "Hub ID"
// @Param        body    body  object{user_id=string}  true  "User payload"
// @Success      204  "Hub successfully deleted"
// @Failure      400  {object}  models.HttpErrorResponse  "Invalid request body"
// @Failure      403  {object}  models.HttpErrorResponse  "User is not authenticated or not the hub owner"
// @Failure      404  {object}  models.HttpErrorResponse  "Hub not found"
// @Failure      500  {object}  models.HttpErrorResponse  "Internal server error"
// @Security     CookieAuth
// @Router       /hub/{hub_id} [delete]
func (h *HubAPI) DeleteHubHandler(ctx *gin.Context) {
	log := mod.GetLogger(ctx)
	hub, ok := isValHub(ctx, "hub")
	if !ok {
		log.Error("Invalid hub id", zap.Any("hub", hub))
		ctx.Data(http.StatusNotFound, mod.APP_JSON, mod.HttpError(mod.INVALID_URL_CODE, mod.INVALID_URL_MSG))
		return
	}

	val, ok := ctx.Get("host_and_hub_cache")
	if !ok {
		log.Error("Invalid host_and_hub_cache type")
		ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
		return
	}

	userID, ok := helpers.IsValString(ctx, "user_id")
	if !ok {
		log.Error("user_id is missing")
		ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
		return
	}

	isOwner := false
	switch cache := val.(type) {
	case *HostAndHubs:
		for _, id := range cache.GetHubs(userID) {
			if id == hub.ID {
				isOwner = true
				break
			}
		}
		if !isOwner {
			ctx.Data(http.StatusForbidden, mod.APP_JSON, mod.HttpError(mod.FORBIDDEN_CODE, mod.FORBIDDEN_MSG))
			return
		}

		h.MGR.Delete(hub.ID)
		cache.RemoveHub(userID, hub.ID)
		ctx.Status(http.StatusNoContent)
	default:
		log.Error("Invalid host_and_hub_cache type")
		ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
	}
}

// ListenHandler godoc
// @Summary      Listen to audio stream
// @Description  Subscribes the client to a hub's audio stream and delivers synthesized audio chunks in real-time via chunked transfer encoding. The stream ends when the client disconnects or the hub closes.
// @Tags         hub
// @Produce      audio/mpeg
// @Param        hub_id  path  string  true  "Hub ID"
// @Success      200  "Audio stream delivered as chunked audio/mpeg"
// @Failure      404  {object}  models.HttpErrorResponse  "Invalid hub ID (IsHubIDValid middleware)"
// @Router       /hub/{hub_id}/listen [get]
func (h *HubAPI) ListenHandler(ctx *gin.Context) {
	log := mod.GetLogger(ctx)
	hub, ok := isValHub(ctx, "hub")
	if !ok {
		log.Error("Invalid hub id", zap.Any("hub", hub))
		ctx.Data(http.StatusNotFound, mod.APP_JSON, mod.HttpError(mod.INVALID_URL_CODE, mod.INVALID_URL_MSG))
		return
	}

	consumerID := uuid.New().String()
	consumer := &Consumer{
		ID:   consumerID,
		Send: make(chan []byte, 128),
	}

	hub.AddConsumer(consumer)
	defer hub.RemoveConsumer(consumerID)

	ctx.Header("Content-Type", "audio/mpeg")
	ctx.Header("Transfer-Encoding", "chunked")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Status(http.StatusOK)

	clientGone := ctx.Request.Context().Done()

	log.Debug("Audio stream started", zap.String("hub_id", hub.ID), zap.String("consumer_id", consumerID))
	ctx.Stream(func(w io.Writer) bool {
		select {
		case chunk, ok := <-consumer.Send:
			if !ok {
				return false
			}
			_, err := w.Write(chunk)
			return err == nil

		case <-clientGone:
			return false
		}
	})
	log.Debug("Audio stream ended", zap.String("hub_id", hub.ID), zap.String("consumer_id", consumerID))
}

func (h *HubAPI) FishSDK(ctx *gin.Context) {
	ctx.Set("voice_agent_builder", &BuildHolder{
		client: fishaudio.NewClient(
			fishaudio.WithAPIKey(h.Cfg.FishAudioAPIKey),
			fishaudio.WithBaseURL(h.Cfg.FishAudioBaseURL),
		).TTS,
	})
	ctx.Next()
}

func (h *HubAPI) OpenAISDK(ctx *gin.Context) {
	ctx.Set("voice_agent_builder", &OpenAIBuilder{
		client: openai.NewClient(
			option.WithAPIKey(h.Cfg.OpenAIAPIKey),
			option.WithBaseURL(h.Cfg.OpenAIBaseURL),
		),
	})
	ctx.Next()
}

type Closer interface {
	Close() error
}

func doCloser(cl Closer, log *zap.Logger) {
	err := cl.Close()
	if err != nil {
		log.Error("Failed to close reader", zap.Error(err))
	}
}

func (h *HubAPI) WsUpgrader(ctx *gin.Context) {
	ctx.Set("ws_upgrader", &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return origin == "https://bogdanantonovich.com" || origin == "https://www.bogdanantonovich.com"
		},
	})
	ctx.Next()
}

// PublishHandler godoc
// @Summary      Publish audio stream via WebSocket
// @Description  Upgrades to WebSocket, receives audio chunks, transcribes via Deepgram, processes via Groq, and synthesizes speech. All operations run concurrently. Requires a valid user and hub in context.
// @Tags         hub
// @Param        hub_id  path   string  true  "Hub ID"
// @Param        lang    query  string  true  "Transcription language code (e.g. en, ru)"
// @Param        file_id query  string  true  "File ID to the voice reference"
// @Success      101  "WebSocket upgrade successful"
// @Failure      401  {object}  models.HttpErrorResponse  "Missing or invalid auth cookies"
// @Failure      404  {object}  models.HttpErrorResponse  "Invalid user ID or hub"
// @Failure      500  {object}  models.HttpErrorResponse  "Internal server error"
// @Security     CookieAuth
// @Router       /hub/{hub_id}/publish [get]
func (h *HubAPI) PublishHandler(ctx *gin.Context) {
	log := mod.GetLogger(ctx)
	transcription := NewStringChanBuf(1)
	tokens := NewStringChanBuf(1)
	groqErrors := NewErrorChanBuf(1)
	deepgramErrors := NewErrorChanBuf(1)

	upgrader, ok := mod.GetWSUpgrader(ctx, log)
	if !ok {
		ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
		return
	}

	userID, ok := helpers.IsValString(ctx, "user_id")
	if !ok {
		log.Error("User id is invalid", zap.String("user_id", userID))
		ctx.Data(http.StatusNotFound, mod.APP_JSON, mod.HttpError(mod.INVALID_URL_CODE, mod.INVALID_URL_MSG))
		return
	}

	val, ok := ctx.Get("host_and_hub_cache")
	if !ok {
		log.Error("Invalid host_and_hub_cache type")
		ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
		return
	}

	hub, ok := isValHub(ctx, "hub")
	if !ok {
		log.Error("Hub is invalid", zap.Any("hub", hub))
		ctx.Data(http.StatusNotFound, mod.APP_JSON, mod.HttpError(mod.INVALID_URL_CODE, mod.INVALID_URL_MSG))
		return
	}

	isOwner := false
	switch cache := val.(type) {
	case *HostAndHubs:
		for _, id := range cache.GetHubs(userID) {
			if id == hub.ID {
				isOwner = true
				break
			}
		}
		if !isOwner {
			log.Debug("user is not the owner", zap.String("user_id", userID), zap.String("hub_id", hub.ID))
			ctx.Data(http.StatusForbidden, mod.APP_JSON, mod.HttpError(mod.FORBIDDEN_CODE, mod.FORBIDDEN_MSG))
			return
		}
	default:
		log.Error("Invalid host_and_hub_cache type")
		ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
		return
	}

	fileID := ctx.Query("file_id")
	if fileID == "" {
		log.Error("File ID is missing")
		ctx.Data(http.StatusBadRequest, mod.APP_JSON, mod.HttpError(mod.INVALID_URL_CODE, mod.INVALID_URL_MSG))
		return
	}

	var agent VoiceAgent
	if builder, ok := ctx.Get("voice_agent_builder"); !ok {
		log.Error("voice_agent_builder not found in context")
		ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
		return
	} else {
		switch bldr := builder.(type) {
		case VoiceAgentBuilder:
			bldr.SetHub(hub)
			bldr.SetTokens(tokens)
			bldr.SetLogger(log)
			agent = bldr.Get()
		default:
			log.Error("voice_agent_builder is not a VoiceAgentBuilder")
			ctx.Data(http.StatusInternalServerError, mod.APP_JSON, mod.HttpError(mod.INTERNAL_ERROR_CODE, mod.INTERNAL_ERROR_MSG))
			return
		}
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}
	defer doCloser(conn, log)

	log.Debug("Audio publishing started", zap.String("user_id", userID), zap.String("hub_id", hub.ID))

	pr, pw := io.Pipe()

	g, gctx := errgroup.WithContext(ctx.Request.Context())

	go func() {
		<-gctx.Done()
		pw.CloseWithError(gctx.Err())
	}()

	deepgram := Deepgram{
		ApiKey:  h.Cfg.DeepgramAPIKey,
		BaseURL: h.Cfg.DeepgramBaseURL,
		Options: interfaces.LiveTranscriptionOptions{
			Model:       h.Cfg.DeepgramModel,
			Language:    ctx.Query("lang"),
			Channels:    1,
			Endpointing: "true",
			Numerals:    true,
			Punctuate:   true,
		},
		transcription: transcription,
		errors:        deepgramErrors,
		log:           log,
		ctx:           gctx,
	}

	groq := Groq{
		ApiKey:        h.Cfg.GroqAPIKey,
		Model:         h.Cfg.GroqModel,
		BaseURL:       h.Cfg.GroqBaseURL,
		transcription: transcription,
		errors:        groqErrors,
		tokens:        tokens,
		log:           log,
	}

	g.Go(func() error {
		defer doCloser(pw, log)
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					return nil
				}
				log.Error("WebSocket read error", zap.Error(err))
				return err
			}
			if msgType == websocket.BinaryMessage {
				if _, err := pw.Write(msg); err != nil {
					return err
				}
			}
		}
	})

	g.Go(func() error { defer transcription.Close(); return deepgram.do(pr) })
	g.Go(func() error { defer tokens.Close(); return groq.do(gctx) })
	g.Go(func() error { return agent.Do(gctx) })

	if err := g.Wait(); err != nil {
		transcription.Close()
		deepgramErrors.Close()
		groqErrors.Close()
		tokens.Close()
		log.Error("Publishing pipeline error", zap.Error(err))
		return
	}

	log.Debug("Audio publishing ended", zap.String("user_id", userID), zap.String("hub_id", hub.ID))
}

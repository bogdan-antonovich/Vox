package hub

import (
	"context"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"go.uber.org/zap"
)

// translationRequestTimeout bounds a single translation request so a stalled
// call cannot block the pipeline indefinitely.
const translationRequestTimeout = 10 * time.Second

type Groq struct {
	ApiKey        string
	Model         string
	BaseURL       string
	SourceLang    string
	OutputLang    string
	transcription *StringChan
	tokens        *StringChan
	errors        *ErrorChan
	log           *zap.Logger
}

var langNames = map[string]string{
	"en": "English", "ru": "Russian", "uk": "Ukrainian",
	"de": "German", "fr": "French", "es": "Spanish",
	"zh": "Chinese", "ja": "Japanese", "ar": "Arabic",
	"pt": "Portuguese", "it": "Italian", "ko": "Korean",
	"pl": "Polish", "nl": "Dutch", "tr": "Turkish",
}

func langName(code string) string {
	if name, ok := langNames[code]; ok {
		return name
	}
	return code
}

// func (g *Groq) handleStream(ctx context.Context, stream *ssestream.Stream[openai.ChatCompletionChunk]) error {
// 	g.log.Debug("Groq.handleStream", zap.Bool("ctx_is_nil", ctx == nil), zap.Bool("stream_is_nil", stream == nil))
// 	for stream.Next() {
// 		chunk := stream.Current()
// 		if len(chunk.Choices) > 0 {
// 			content := chunk.Choices[0].Delta.Content
// 			if content != "" {
// 				g.log.Debug("Translation of transcript received", zap.String("content", content))
// 				select {
// 				case g.tokens.Ch <- content:
// 				case <-ctx.Done():
// 					return ctx.Err()
// 				}
// 			}
// 		}
// 	}
// 	return nil
// }

func (g *Groq) do(ctx context.Context) (err error) {
	g.log.Debug("Groq.do", zap.Bool("ctx_is_nil", ctx == nil))
	client := openai.NewClient(
		option.WithAPIKey(g.ApiKey),
		option.WithBaseURL(g.BaseURL),
	)
	for {
		select {
		case transcript, ok := <-g.transcription.Ch:
			if !ok {
				g.log.Debug("Groq transcription channel closed")
				return nil
			}
			g.log.Debug("Groq got transcript", zap.String("transcript", transcript))
			reqCtx, cancel := context.WithTimeout(ctx, translationRequestTimeout)
			resp, err := client.Chat.Completions.New(reqCtx, openai.ChatCompletionNewParams{
				Model: g.Model,
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.SystemMessage("You are a translator. Translate the following " + langName(g.SourceLang) + " text to " + langName(g.OutputLang) + ". Output only the translation, nothing else."),
					openai.UserMessage(transcript),
				},
			})
			cancel()
			if err != nil {
				// Skip this segment instead of tearing down the pipeline so a
				// single stalled/failed translation doesn't stop the broadcast.
				if ctx.Err() != nil {
					return ctx.Err()
				}
				g.log.Error("Groq completion error", zap.Error(err))
				continue
			}
			if len(resp.Choices) > 0 {
				content := resp.Choices[0].Message.Content
				if content != "" {
					g.log.Debug("Groq translation complete", zap.String("content", content))
					select {
					case g.tokens.Ch <- content:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

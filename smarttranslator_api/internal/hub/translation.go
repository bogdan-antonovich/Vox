package hub

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"go.uber.org/zap"
)

type Groq struct {
	ApiKey        string
	Model         string
	BaseURL       string
	TargetLang    string
	transcription *StringChan
	tokens        *StringChan
	errors        *ErrorChan
	log           *zap.Logger
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
			resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
				Model: g.Model,
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.SystemMessage("You are a translator. Translate the following " + g.TargetLang + " text to English. Output only the translation, nothing else."),
					openai.UserMessage(transcript),
				},
			})
			if err != nil {
				g.log.Error("Groq completion error", zap.Error(err))
				return err
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

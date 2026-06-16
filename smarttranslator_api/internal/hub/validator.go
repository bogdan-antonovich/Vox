package hub

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"go.uber.org/zap"
)

// Validator sits between speech-to-text (Deepgram) and the translator (Groq) in
// the broadcast pipeline. Its job is to make sure Groq only ever translates
// *complete thoughts* in the source language, rather than the half-sentence
// fragments Deepgram emits as it transcribes:
//
//   - It accumulates raw transcript text from the input channel into an internal
//     buffer (the "save").
//   - On every incoming chunk it asks an LLM to split the accumulated text into
//     complete thoughts plus a trailing incomplete remainder.
//   - Each complete thought is emitted, in original order (FIFO), to the output
//     channel for translation. A long chunk containing several complete thoughts
//     is therefore split and delivered as separate, ordered pieces.
//   - The incomplete remainder is kept in the buffer until the rest of it
//     arrives, so the translator never receives a half-finished sentence.
//   - When the input channel closes, whatever is left in the buffer is flushed
//     as a final thought.
type Validator struct {
	ApiKey    string
	Model     string
	BaseURL   string
	Lang      string      // source language of the transcript, for the prompt
	tokens    *StringChan // input: raw transcript from Deepgram
	validated *StringChan // output: complete source-language thoughts for Groq
	log       *zap.Logger
}

// validatorResult is the JSON contract we ask the segmentation model to honour.
type validatorResult struct {
	Complete  []string `json:"complete"`
	Remainder string   `json:"remainder"`
}

func validatorSystemPrompt(lang string) string {
	return "You are a sentence-segmentation assistant for a real-time speech " +
		"translation pipeline. The text is in " + langName(lang) + ". You receive a " +
		"chunk of text that may contain zero, one, or several complete thoughts " +
		"(sentences), optionally followed by an incomplete trailing fragment. " +
		"Return STRICT JSON with exactly two fields: \"complete\" (an array of the " +
		"complete thoughts in their original order) and \"remainder\" (the trailing " +
		"incomplete fragment, or an empty string if the text ends on a complete " +
		"thought). Do not translate, rephrase, summarize, add or drop any words; " +
		"preserve the original wording and order exactly. Output only the JSON object."
}

// extractJSON returns the substring spanning the first '{' to the last '}', so a
// response wrapped in markdown fences or prose still parses. If no braces are
// present the original string is returned unchanged.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end < start {
		return s
	}
	return s[start : end+1]
}

func (v *Validator) emit(ctx context.Context, thought string) error {
	select {
	case v.validated.Ch <- thought:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// segment asks the model to split text into complete thoughts and a remainder.
// If the model does not return parseable JSON it falls back to treating the
// whole text as a single complete thought, so audio keeps flowing rather than
// stalling in the buffer forever.
func (v *Validator) segment(ctx context.Context, client openai.Client, text string) (complete []string, remainder string, err error) {
	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: v.Model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(validatorSystemPrompt(v.Lang)),
			openai.UserMessage(text),
		},
	})
	if err != nil {
		v.log.Error("Validator completion error", zap.Error(err))
		return nil, "", err
	}
	if len(resp.Choices) == 0 {
		v.log.Warn("Validator got no choices, keeping text buffered")
		return nil, text, nil
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	var result validatorResult
	if jsonErr := json.Unmarshal([]byte(extractJSON(content)), &result); jsonErr != nil {
		v.log.Warn("Validator response is not JSON, forwarding whole text as one thought",
			zap.String("content", content), zap.Error(jsonErr))
		return []string{text}, "", nil
	}

	v.log.Debug("Validator segmented text",
		zap.Strings("complete", result.Complete),
		zap.String("remainder", result.Remainder))
	return result.Complete, result.Remainder, nil
}

func (v *Validator) do(ctx context.Context) error {
	v.log.Debug("Validator.do", zap.Bool("ctx_is_nil", ctx == nil))
	client := openai.NewClient(
		option.WithAPIKey(v.ApiKey),
		option.WithBaseURL(v.BaseURL),
	)

	var buffer strings.Builder
	for {
		select {
		case chunk, ok := <-v.tokens.Ch:
			if !ok {
				v.log.Debug("Validator input channel closed, flushing buffer")
				rest := strings.TrimSpace(buffer.String())
				if rest != "" {
					if err := v.emit(ctx, rest); err != nil {
						return err
					}
				}
				return nil
			}

			chunk = strings.TrimSpace(chunk)
			if chunk == "" {
				continue
			}
			if buffer.Len() > 0 {
				buffer.WriteString(" ")
			}
			buffer.WriteString(chunk)

			text := strings.TrimSpace(buffer.String())
			complete, remainder, err := v.segment(ctx, client, text)
			if err != nil {
				return err
			}

			for _, thought := range complete {
				thought = strings.TrimSpace(thought)
				if thought == "" {
					continue
				}
				if err := v.emit(ctx, thought); err != nil {
					return err
				}
			}

			buffer.Reset()
			buffer.WriteString(strings.TrimSpace(remainder))
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Prompt bodies and answer decoders shared by the provider adapters (Gemini, OpenAI-compatible): the instructions
// are the same whatever model answers, and so is the JSON they ask for.

// vocabularyLines renders the NLU vocabulary: one "key | type | label | allowed values" line per slot.
func vocabularyLines(vocab []VocabEntry) string {
	var vb strings.Builder
	for _, v := range vocab {
		vb.WriteString(v.Key + " | " + v.Type + " | " + v.Label)
		switch v.Type {
		case "single", "items":
			codes := make([]string, len(v.Values))
			for i, x := range v.Values {
				codes[i] = x.Code + "=" + x.Label
			}
			vb.WriteString(" | " + strings.Join(codes, ", "))
		case "number", "integer":
			fmt.Fprintf(&vb, " | %g–%g %s", v.Min, v.Max, v.Unit)
		case "text":
			fmt.Fprintf(&vb, " | ≤ %d characters", v.MaxLen)
		case "time":
			vb.WriteString(" | HH:MM")
		}
		vb.WriteByte('\n')
	}
	return vb.String()
}

// parseLogPrompt is the user turn of the NLU call.
func parseLogPrompt(req LogParseRequest) string {
	return "LANGUAGE: " + req.Language + "\nVOCABULARY:\n" + vocabularyLines(req.Vocabulary) + "TEXT:\n<<<\n" + req.Text + "\n>>>"
}

// stripJSONFence removes a ```json … ``` wrapper some models put around a JSON answer.
func stripJSONFence(text string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(text), "```json"), "```"))
}

// decodeCandidates reads the NLU answer.
func decodeCandidates(text string) ([]Candidate, error) {
	var parsed struct {
		Items []struct {
			Key          string   `json:"key"`
			Value        any      `json:"value"`
			Confidence   float64  `json:"confidence"`
			Alternatives []string `json:"alternatives"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(stripJSONFence(text)), &parsed); err != nil {
		return nil, fmt.Errorf("%w: unparsable answer", ErrUpstream)
	}
	out := make([]Candidate, 0, len(parsed.Items))
	for _, it := range parsed.Items {
		out = append(out, Candidate{Key: it.Key, Value: it.Value, Confidence: it.Confidence, Alternatives: it.Alternatives})
	}
	return out, nil
}

// extractPrompt is the text part of an extraction call (the document travels next to it).
func extractPrompt(req ExtractRequest) string {
	var sb strings.Builder
	sb.WriteString("DOCUMENT TYPE: " + req.Schema.Name + "\nLANGUAGE: " + req.Language + "\n")
	if req.Hint != "" {
		sb.WriteString("CONTEXT: " + req.Hint + "\n")
	}
	if len(req.Schema.Fields) > 0 {
		sb.WriteString("FIELDS (key | type | description | allowed values):\n")
		schemaLines(&sb, req.Schema.Fields)
	}
	if len(req.Schema.Items) > 0 {
		sb.WriteString("ITEMS (key | type | description | allowed values):\n")
		schemaLines(&sb, req.Schema.Items)
	}
	return sb.String()
}

// decodeExtraction reads an extraction answer.
func decodeExtraction(text string) (Extraction, error) {
	var parsed struct {
		Fields []gExtracted   `json:"fields"`
		Items  [][]gExtracted `json:"items"`
	}
	if err := json.Unmarshal([]byte(stripJSONFence(text)), &parsed); err != nil {
		return Extraction{}, fmt.Errorf("%w: unparsable answer", ErrUpstream)
	}
	conv := func(in []gExtracted) []ExtractedField {
		out := make([]ExtractedField, len(in))
		for i, f := range in {
			out[i] = ExtractedField(f)
		}
		return out
	}
	out := Extraction{Fields: conv(parsed.Fields), Items: make([][]ExtractedField, 0, len(parsed.Items))}
	for _, r := range parsed.Items {
		out.Items = append(out.Items, conv(r))
	}
	return out, nil
}

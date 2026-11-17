package ai

// Voice → autopilot: the gateway's speech endpoint (multipart /audio/transcriptions), the script call and the
// deterministic fake.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAICompat_Transcribe(t *testing.T) {
	var path, auth, model, lang, fileName, fileType string
	var file []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, auth = r.URL.Path, r.Header.Get("Authorization")
		require.NoError(t, r.ParseMultipartForm(1<<20))
		model, lang = r.FormValue("model"), r.FormValue("language")
		f, h, err := r.FormFile("file")
		require.NoError(t, err)
		fileName, fileType = h.Filename, h.Header.Get("Content-Type")
		file, _ = io.ReadAll(f)
		_, _ = io.WriteString(w, `{"text":"  دلم درد دارم  ","usage":{"type":"tokens","input_tokens":120,"output_tokens":9,"input_token_details":{"audio_tokens":110,"text_tokens":10}}}`)
	}))
	t.Cleanup(srv.Close)
	o := NewOpenAICompat(OpenAICompatConfig{Provider: ProviderArvan, BaseURL: srv.URL + "/v1/", APIKey: "secret-key",
		AuthScheme: "apikey", Model: "default-model"}, srv.Client())
	ctx := withRoute(context.Background(), Route{Provider: ProviderArvan, Model: "speech-model"})
	got, u, err := o.Transcribe(ctx, TranscribeRequest{Audio: Audio{Data: []byte("opus-bytes"), MIME: "audio/webm"}, Language: "fa"})
	require.NoError(t, err)
	assert.Equal(t, Transcript{Text: "دلم درد دارم", Language: "fa"}, got)
	assert.Equal(t, "/v1/audio/transcriptions", path)
	assert.Equal(t, "apikey secret-key", auth)
	assert.Equal(t, "speech-model", model)
	assert.Equal(t, "fa", lang)
	assert.Equal(t, "audio.webm", fileName)
	assert.Equal(t, "audio/webm", fileType)
	assert.Equal(t, "opus-bytes", string(file))
	assert.Equal(t, "speech-model", u.Model)
	assert.Equal(t, ProviderArvan, u.Provider)
	assert.Equal(t, [3]int{120, 9, 110}, [3]int{u.InputTokens, u.OutputTokens, u.AudioTokens}, "gpt-4o-transcribe style usage")
}

func TestOpenAICompat_TranscribeFailureIsUpstream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"echo of the request"}`)
	}))
	t.Cleanup(srv.Close)
	o := NewOpenAICompat(OpenAICompatConfig{Provider: ProviderArvan, BaseURL: srv.URL, APIKey: "k"}, srv.Client())
	_, _, err := o.Transcribe(context.Background(), TranscribeRequest{Audio: Audio{Data: []byte("x"), MIME: "audio/ogg"}})
	require.ErrorIs(t, err, ErrUpstream)
	assert.NotContains(t, err.Error(), "echo", "an error body is never read into the error")
}

func TestOpenAICompat_PilotScript(t *testing.T) {
	o, g := newCompat(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, completion("```json\n{\"version\":1,\"steps\":[]}\n```", "stop"))
	})
	out, _, err := o.PilotScript(context.Background(), PilotRequest{Transcript: "نفخ دارم", Manifest: []byte(`{"targets":[]}`), Language: "fa", Today: "2026-11-16"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"version":1,"steps":[]}`, out)
	msgs, _ := g.bodies[0]["messages"].([]any)
	require.Len(t, msgs, 2)
	user, _ := msgs[1].(map[string]any)
	assert.Contains(t, user["content"], "نفخ دارم")
	assert.Contains(t, user["content"], `{"targets":[]}`)
	assert.Contains(t, user["content"], "TODAY: 2026-11-16")
}

const pilotManifest = `{"targets":[
 {"id":"log.param:symptoms.digestive","kind":"multi","label":"گوارش","options":[{"value":"bloating","label":"نفخ"},{"value":"nausea","label":"تهوع"}]},
 {"id":"log.param:bleeding.flow","kind":"single","label":"شدت","options":[{"value":"light","label":"کم"},{"value":"heavy","label":"زیاد"}]},
 {"id":"log.param:bleeding.spotting","kind":"bool","label":"لکه"},
 {"id":"log.param:note.text","kind":"text","label":"یادداشت"}]}`

func TestFake_PilotScript(t *testing.T) {
	f := NewFake()
	out, _, err := f.PilotScript(context.Background(), PilotRequest{Transcript: "امروز نفخ دارم و لکه، خونریزی زیاد", Manifest: []byte(pilotManifest)})
	require.NoError(t, err)
	assert.JSONEq(t, `{"version":1,"steps":[
		{"do":"set","target":"log.param:symptoms.digestive","value":["bloating"]},
		{"do":"set","target":"log.param:bleeding.flow","value":"heavy"},
		{"do":"set","target":"log.param:bleeding.spotting","value":true}]}`, out)

	out, _, err = f.PilotScript(context.Background(), PilotRequest{Transcript: "هوا خوبه", Manifest: []byte(pilotManifest)})
	require.NoError(t, err)
	var empty struct {
		Steps []any `json:"steps"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &empty))
	assert.Empty(t, empty.Steps)

	_, _, err = f.PilotScript(context.Background(), PilotRequest{Transcript: "x", Manifest: []byte("not json")})
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestClient_PilotScript(t *testing.T) {
	rec := &spyRecorder{}
	c := NewClientWith(Options{Provider: "fake", Scripter: NewFake(), Recorder: rec})
	_, err := c.PilotScript(context.Background(), FeatureVoiceLog, PilotRequest{Transcript: "نفخ", Manifest: []byte(pilotManifest)})
	require.NoError(t, err)
	require.Len(t, rec.got, 1)
	assert.Equal(t, OpPilot, rec.got[0].Op)
	assert.Equal(t, FeatureVoiceLog, rec.got[0].Feature)

	_, err = NewClient("fake", NewFake(), NewFake(), nil).PilotScript(context.Background(), FeatureVoiceLog, PilotRequest{})
	require.ErrorIs(t, err, ErrUnavailable, "a provider that cannot write scripts")
	var none *Client
	_, err = none.PilotScript(context.Background(), FeatureVoiceLog, PilotRequest{})
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestVoiceSlots(t *testing.T) {
	log, ok := SpecOf(FeatureVoiceLog)
	require.True(t, ok)
	assert.Equal(t, []Capability{CapJSON}, log.Needs)
	assert.False(t, log.Internal)
	stt, ok := SpecOf(FeatureVoiceSTT)
	require.True(t, ok)
	assert.Equal(t, []Capability{CapAudio}, stt.Needs)
	assert.True(t, stt.Internal, "the speech slot is never listed to users")
}

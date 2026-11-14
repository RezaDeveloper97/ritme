package ai

import (
	"context"
	"math"
	"slices"
	"sort"
	"sync"
)

// Model routing (goose 00125): which model serves which feature is admin data (ai_features → ai_models), not
// configuration. A feature declares itself once in code (RegisterFeature: its key and the capability its calls
// need); the admin then picks, per feature, a model that has that capability, a fallback model and the credit
// cost of one use. The Client asks its Router on every call:
//
//	feature → Routes (primary, then fallback) → the provider adapter of each Route → the call
//
// The model name travels to the adapter on the context (routeModel); the adapter's configured default model is
// used when there is no Router (tests, tools) — the behaviour before routing existed.

// Capability is what a model can do. A feature's model must have the capability its calls need.
type Capability string

// Capabilities.
const (
	CapChat   Capability = "chat"   // streamed text chat
	CapVision Capability = "vision" // images / documents in
	CapAudio  Capability = "audio"  // speech in
	CapJSON   Capability = "json"   // dependable structured (JSON) answers
)

// Capabilities lists every capability (admin validation).
func Capabilities() []Capability { return []Capability{CapChat, CapVision, CapAudio, CapJSON} }

// Provider ids of real adapters (ai_models.provider). The fake is never a catalogue row: it replaces every route
// while AI_PROVIDER=fake.
const (
	ProviderGemini = "gemini"
	ProviderArvan  = "arvan" // ArvanCloud AI gateway (OpenAI-compatible, openai_compat.go)
)

// Providers lists the adapters a catalogue model may name.
func Providers() []string { return []string{ProviderGemini, ProviderArvan} }

// FeatureSpec is a feature slot declared in code.
type FeatureSpec struct {
	Key Feature
	// Needs are the capabilities a model must have to serve the feature.
	Needs []Capability
}

var (
	registryMu sync.RWMutex
	registry   = map[Feature]FeatureSpec{}
)

// RegisterFeature declares an AI feature slot. Call it from a package-level var or init of the feature's package;
// the catalogue creates the slot's ai_features row at start-up, where the admin assigns its model and credit
// cost. Registering the same key twice with different needs panics (a copy-paste must not hijack a slot).
func RegisterFeature(key Feature, needs ...Capability) Feature {
	registryMu.Lock()
	defer registryMu.Unlock()
	if old, ok := registry[key]; ok && !slices.Equal(old.Needs, needs) {
		panic("ai: feature " + string(key) + " registered twice")
	}
	registry[key] = FeatureSpec{Key: key, Needs: needs}
	return key
}

// RegisteredFeatures are the declared slots, by key.
func RegisteredFeatures() []FeatureSpec {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]FeatureSpec, 0, len(registry))
	for _, s := range registry {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// SpecOf is the declared slot of key.
func SpecOf(key Feature) (FeatureSpec, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	s, ok := registry[key]
	return s, ok
}

// The slots of the features that existed before routing.
var (
	_ = RegisterFeature(FeatureVoiceLog, CapAudio, CapJSON)
	_ = RegisterFeature(FeatureLabAnalysis, CapVision, CapJSON)
	_ = RegisterFeature(FeatureAssistant, CapChat)
	_ = RegisterFeature(FeatureDocExtract, CapVision, CapJSON)
)

// Route is one model that may serve a call.
type Route struct {
	Provider string // ProviderGemini | ProviderArvan
	Model    string // the name sent upstream
	// Prices in milli-USD per 1M tokens (ai_models); Priced false = use the AI_PRICES table.
	Priced           bool
	InputPriceMilli  uint32
	OutputPriceMilli uint32
	AudioPriceMilli  uint32
}

// cost is the estimated cost of u on this route in micro-USD (rounded up): tokens × milli-USD per 1M tokens / 1000.
func (r Route) cost(u Usage) uint64 {
	audio := max(0, min(u.AudioTokens, u.InputTokens))
	text := max(0, u.InputTokens-audio)
	c := float64(text)*float64(r.InputPriceMilli) + float64(audio)*float64(r.AudioPriceMilli) +
		float64(max(0, u.OutputTokens))*float64(r.OutputPriceMilli)
	return uint64(math.Ceil(c / 1000))
}

// Router resolves the models of a feature, best first (primary, then fallback). No routes = the feature has no
// usable model (ErrUnavailable).
type Router interface {
	Routes(ctx context.Context, feature Feature) ([]Route, error)
}

type routeKey struct{}

func withRoute(ctx context.Context, r Route) context.Context {
	return context.WithValue(ctx, routeKey{}, r)
}

func routeFrom(ctx context.Context) (Route, bool) {
	r, ok := ctx.Value(routeKey{}).(Route)
	return r, ok
}

// routeModel is the model an adapter must call: the route's, else its own configured default.
func routeModel(ctx context.Context, fallback string) string {
	if r, ok := routeFrom(ctx); ok && r.Model != "" {
		return r.Model
	}
	return fallback
}

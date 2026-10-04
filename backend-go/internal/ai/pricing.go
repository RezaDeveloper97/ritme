package ai

import (
	"math"

	"github.com/ritme/backend-go/internal/platform/config"
)

// Pricer estimates the cost of a call from the AI_PRICES table (config.AI.Prices, USD per 1M tokens). With rates
// per million tokens, tokens × rate is exactly micro-USD, so costs are integers of micro-USD (cost_micros).
//
// The fake is always free. A model missing from the table is priced at the table's highest rate of each kind:
// an unpriced model must never look free to the daily cap.
type Pricer struct {
	prices map[string]config.AIPrice
	max    config.AIPrice
}

// NewPricer builds a pricer over prices (nil / empty → every real call priced at 0; tests only).
func NewPricer(prices map[string]config.AIPrice) Pricer {
	p := Pricer{prices: prices}
	for _, r := range prices {
		p.max.InputPerMTok = math.Max(p.max.InputPerMTok, r.InputPerMTok)
		p.max.OutputPerMTok = math.Max(p.max.OutputPerMTok, r.OutputPerMTok)
		p.max.AudioPerMTok = math.Max(p.max.AudioPerMTok, r.AudioPerMTok)
	}
	return p
}

// Cost is the estimated cost of u in micro-USD (rounded up).
func (p Pricer) Cost(u Usage) uint64 {
	if u.Provider == config.AIProviderFake {
		return 0
	}
	r, ok := p.prices[u.Model]
	if !ok {
		r = p.max
	}
	audio := max(0, min(u.AudioTokens, u.InputTokens))
	text := max(0, u.InputTokens-audio)
	c := float64(text)*r.InputPerMTok + float64(audio)*r.AudioPerMTok + float64(max(0, u.OutputTokens))*r.OutputPerMTok
	return uint64(math.Ceil(c))
}

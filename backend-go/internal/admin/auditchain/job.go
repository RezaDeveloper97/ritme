package auditchain

import (
	"context"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ritme/backend-go/internal/platform/metrics"
	"github.com/ritme/backend-go/internal/platform/queue"
)

// JobVerify is the nightly verification (docs/engine-v6/jobs.md).
const (
	JobVerify  = "admin.audit_verify"
	VerifyCron = "40 3 * * *"
)

var (
	chainBroken = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "admin_audit_chain_broken",
		Help: "1 when the last admin.audit_verify run found a break in the admin audit hash chain, else 0.",
	})
	verifyRuns = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "admin_audit_chain_verify_total",
		Help: "admin.audit_verify runs by result (ok, broken, error).",
	}, []string{"result"})
)

func init() { metrics.Registry().MustRegister(chainBroken, verifyRuns) }

// RegisterJobs registers admin.audit_verify on q.
func RegisterJobs(q *queue.Queue, c *Chain, logger *slog.Logger) error {
	return q.RegisterPeriodic(queue.Job{
		Type: JobVerify, Priority: queue.Low, MaxTries: 1, Timeout: 10 * time.Minute,
	}, VerifyJob(c, logger), VerifyCron, queue.Unique(23*time.Hour))
}

// VerifyJob is the admin.audit_verify handler: run genesis if it has not run, verify the whole chain, set the
// admin_audit_chain_broken gauge and write the head (id + hash) to the structured log as the off-database anchor.
// Idempotent: it only reads, apart from the one-off genesis. Rows' content never reaches the log.
func VerifyJob(c *Chain, logger *slog.Logger) queue.Handler {
	return func(ctx context.Context, _ []byte) error {
		_, err := RunVerify(ctx, c, logger)
		return err
	}
}

// RunVerify is one verification run with its metric and log lines.
func RunVerify(ctx context.Context, c *Chain, logger *slog.Logger) (Report, error) {
	if err := c.EnsureGenesis(ctx); err != nil {
		verifyRuns.WithLabelValues("error").Inc()
		return Report{}, err
	}
	rep, err := c.Verify(ctx, 0)
	if err != nil {
		verifyRuns.WithLabelValues("error").Inc()
		return rep, err
	}
	anchor := []any{slog.Uint64("head_id", rep.HeadID), slog.String("head_hash", rep.HeadHash),
		slog.Int("checked", rep.Checked)}
	if rep.Break != nil {
		chainBroken.Set(1)
		verifyRuns.WithLabelValues("broken").Inc()
		logger.ErrorContext(ctx, "admin audit chain broken", append(anchor, slog.String("alert", "admin_audit_chain_broken"),
			slog.Uint64("break_id", rep.Break.ID), slog.String("reason", rep.Break.Reason))...)
		return rep, nil
	}
	chainBroken.Set(0)
	verifyRuns.WithLabelValues("ok").Inc()
	logger.InfoContext(ctx, "admin audit chain anchor", anchor...)
	return rep, nil
}

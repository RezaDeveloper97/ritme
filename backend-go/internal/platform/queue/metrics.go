package queue

import (
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ritme/backend-go/internal/platform/metrics"
)

// Queue metrics on the process registry (served by the private METRICS_ADDR listener):
//
//	asynq_queue_size{queue,state}             tasks per state, read from asynq.Inspector at scrape time
//	asynq_archived_total{queue}               tasks this process archived after their last try (counter)
//	asynq_task_duration_seconds{type,result}  one try of a job (result ok | error), both modes
var (
	taskDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "asynq_task_duration_seconds",
		Help:    "Duration of one try of a background job by type and result (ok, error).",
		Buckets: []float64{.005, .01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120, 300, 600},
	}, []string{"type", "result"})

	archivedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "asynq_archived_total",
		Help: "Background jobs archived by this process after exhausting their tries, by queue.",
	}, []string{"queue"})

	queueSizeDesc = prometheus.NewDesc("asynq_queue_size",
		"Tasks in an asynq queue by state (pending, active, scheduled, retry, archived, completed).",
		[]string{"queue", "state"}, nil)

	inspected = &inspectorCollector{queues: map[string]*asynq.Inspector{}}
)

func init() {
	metrics.Registry().MustRegister(taskDuration, archivedTotal, inspected)
}

func observeTask(jobType string, err error, d time.Duration) {
	result := "ok"
	if err != nil {
		result = "error"
	}
	taskDuration.WithLabelValues(jobType, result).Observe(d.Seconds())
}

func incArchived(queue string) { archivedTotal.WithLabelValues(queue).Inc() }

// inspectorCollector reads the queue sizes of every started Queue at scrape time (one Lua call per queue).
type inspectorCollector struct {
	mu     sync.Mutex
	queues map[string]*asynq.Inspector // asynq queue name → inspector
}

func trackQueue(q *Queue) {
	ins := asynq.NewInspectorFromRedisClient(q.rdb)
	inspected.mu.Lock()
	defer inspected.mu.Unlock()
	for name := range q.Queues() {
		inspected.queues[name] = ins
	}
}

func untrackQueue(q *Queue) {
	inspected.mu.Lock()
	defer inspected.mu.Unlock()
	for name := range q.Queues() {
		delete(inspected.queues, name)
	}
}

func (c *inspectorCollector) Describe(ch chan<- *prometheus.Desc) { ch <- queueSizeDesc }

func (c *inspectorCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	snapshot := make(map[string]*asynq.Inspector, len(c.queues))
	for k, v := range c.queues {
		snapshot[k] = v
	}
	c.mu.Unlock()
	for name, ins := range snapshot {
		info, err := ins.GetQueueInfo(name)
		if err != nil {
			continue // a queue no task ever reached does not exist in Redis yet
		}
		for state, n := range map[string]int{
			"pending": info.Pending, "active": info.Active, "scheduled": info.Scheduled,
			"retry": info.Retry, "archived": info.Archived, "completed": info.Completed,
		} {
			ch <- prometheus.MustNewConstMetric(queueSizeDesc, prometheus.GaugeValue, float64(n), name, state)
		}
	}
}

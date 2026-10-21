package queue

// StartSchedulerOnly / StopSchedulerOnly run the scheduler without the worker (tests count enqueued tasks).
func StartSchedulerOnly(q *Queue) error { return q.scheduler.start() }

func StopSchedulerOnly(q *Queue) { q.scheduler.stop() }

package ledger

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EventsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ledger_events_processed_total",
		Help: "Total number of fill events successfully recorded",
	})

	EventsDuplicate = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ledger_events_duplicate_total",
		Help: "Total number of duplicate events skipped",
	})

	RecordEventDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ledger_record_event_duration_seconds",
		Help:    "Time taken to record a ledger event",
		Buckets: prometheus.DefBuckets,
	})
)
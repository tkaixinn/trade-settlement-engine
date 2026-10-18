CREATE TABLE IF NOT EXISTS fill_events_analytics
(
    event_id    String,
    trader_id   String,
    clearing_id String,
    amount      Float64,
    created_at  DateTime DEFAULT now()
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(created_at)
ORDER BY (trader_id, created_at);

-- db/clickhouse/schema.sql
-- Analytics store (ADR-005). Privacy-first: there is NO raw IP and NO full user-agent column
-- anywhere in this schema. Geo/device/browser/os are derived in memory by the redirect
-- service and the IP is discarded before the event is emitted. `visitor_hash` is a
-- daily-rotating salted hash (salt destroyed within 24-48h) so visitors cannot be correlated
-- across days — multi-day unique counts are intentionally not computable (privacy > accuracy).

CREATE DATABASE IF NOT EXISTS linkspire;

-- ─────────────────────────────────────────────────────────────────────────────
-- 1. Raw click events. Retained 30 days (raw_click_events retention class), dropped by TTL.
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS linkspire.clicks_raw
(
    event_time       DateTime,
    event_date       Date MATERIALIZED toDate(event_time),
    domain_id        UInt64,
    code             String,
    workspace_id     UInt64,
    country_code     LowCardinality(String),   -- ISO-3166-1 alpha-2, derived then IP discarded
    region           LowCardinality(String),
    device_class     LowCardinality(String),   -- desktop | mobile | tablet | bot | other
    browser_family   LowCardinality(String),
    os_family        LowCardinality(String),
    referrer_domain  LowCardinality(String),   -- host only; path & query stripped
    source           LowCardinality(String),   -- web | qr | api
    is_bot           UInt8,
    status_served    UInt16,                    -- 301 | 302 | 307 | 308
    visitor_hash     UInt64                     -- daily-salted; NOT stable across days
)
ENGINE = MergeTree
PARTITION BY toYYYYMMDD(event_time)            -- daily partitions → cheap TTL drops
ORDER BY (domain_id, code, event_time)
TTL event_time + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192;

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. Daily aggregates (~25 months = 760 days). AggregatingMergeTree keeps partial
--    aggregate states merged incrementally. Retained by TTL on event_date.
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS linkspire.clicks_daily
(
    event_date       Date,
    domain_id        UInt64,
    code             String,
    workspace_id     UInt64,
    country_code     LowCardinality(String),
    device_class     LowCardinality(String),
    browser_family   LowCardinality(String),
    os_family        LowCardinality(String),
    referrer_domain  LowCardinality(String),
    source           LowCardinality(String),
    clicks           AggregateFunction(count),
    -- per-DAY unique estimate only (salt rotates daily); uniqCombined is memory-efficient.
    unique_visitors  AggregateFunction(uniqCombined(12), UInt64)
)
ENGINE = AggregatingMergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (domain_id, code, event_date, country_code, device_class,
          browser_family, os_family, referrer_domain, source)
TTL event_date + INTERVAL 760 DAY DELETE
SETTINGS index_granularity = 8192;

-- Incrementally maintains clicks_daily as rows land in clicks_raw.
CREATE MATERIALIZED VIEW IF NOT EXISTS linkspire.clicks_daily_mv
TO linkspire.clicks_daily AS
SELECT
    toDate(event_time)          AS event_date,
    domain_id,
    code,
    workspace_id,
    country_code,
    device_class,
    browser_family,
    os_family,
    referrer_domain,
    source,
    countState()                AS clicks,
    uniqCombinedState(visitor_hash) AS unique_visitors
FROM linkspire.clicks_raw
GROUP BY event_date, domain_id, code, workspace_id, country_code,
         device_class, browser_family, os_family, referrer_domain, source;

-- Convenience view returning merged, human-readable aggregates.
CREATE VIEW IF NOT EXISTS linkspire.clicks_daily_final AS
SELECT
    event_date, domain_id, code, workspace_id, country_code, device_class,
    browser_family, os_family, referrer_domain, source,
    countMerge(clicks)                 AS clicks,
    uniqCombinedMerge(unique_visitors) AS unique_visitors
FROM linkspire.clicks_daily
GROUP BY event_date, domain_id, code, workspace_id, country_code, device_class,
         browser_family, os_family, referrer_domain, source;

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. Ingestion from Redpanda/Kafka. The Kafka engine table consumes the topic; a MV copies
--    rows into clicks_raw. Producer emits JSONEachRow (protobuf is also supported via
--    kafka_format='Protobuf' + a schema file if lower bandwidth is required).
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS linkspire.clicks_queue
(
    event_time       DateTime,
    domain_id        UInt64,
    code             String,
    workspace_id     UInt64,
    country_code     String,
    region           String,
    device_class     String,
    browser_family   String,
    os_family        String,
    referrer_domain  String,
    source           String,
    is_bot           UInt8,
    status_served    UInt16,
    visitor_hash     UInt64
)
ENGINE = Kafka
SETTINGS
    kafka_broker_list = 'redpanda:9092',
    kafka_topic_list  = 'click_events',
    kafka_group_name  = 'clickhouse_clicks',
    kafka_format      = 'JSONEachRow',
    kafka_num_consumers = 3,
    kafka_max_block_size = 1048576;

CREATE MATERIALIZED VIEW IF NOT EXISTS linkspire.clicks_queue_mv
TO linkspire.clicks_raw AS
SELECT
    event_time, domain_id, code, workspace_id, country_code, region,
    device_class, browser_family, os_family, referrer_domain, source,
    is_bot, status_served, visitor_hash
FROM linkspire.clicks_queue;

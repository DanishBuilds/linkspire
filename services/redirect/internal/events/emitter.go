// services/redirect/internal/events/emitter.go
package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/linkspire/linkspire/services/redirect/internal/observability"
)

// ClickEvent is the privacy-safe analytics record. JSON field names match the ClickHouse
// JSONEachRow columns (db/clickhouse/schema.sql). It contains NO IP and NO user-agent.
type ClickEvent struct {
	EventTime      int64  `json:"event_time"` // unix seconds
	DomainID       int64  `json:"domain_id"`
	Code           string `json:"code"`
	WorkspaceID    int64  `json:"workspace_id"`
	Country        string `json:"country_code"`
	Region         string `json:"region"`
	DeviceClass    string `json:"device_class"`
	BrowserFamily  string `json:"browser_family"`
	OSFamily       string `json:"os_family"`
	ReferrerDomain string `json:"referrer_domain"`
	Source         string `json:"source"`
	IsBot          uint8  `json:"is_bot"`
	StatusServed   uint16 `json:"status_served"`
	VisitorHash    uint64 `json:"visitor_hash"`
}

// Emitter accepts click events. Implementations MUST be non-blocking on the hot path.
type Emitter interface {
	Emit(ClickEvent)
	Close(ctx context.Context) error
}

// Nop discards events (used when Kafka is disabled, e.g. local dev).
type Nop struct{}

func (Nop) Emit(ClickEvent)              {}
func (Nop) Close(context.Context) error  { return nil }

// Kafka is a fire-and-forget emitter with a bounded ring buffer. A full buffer DROPS the
// event (and increments a metric) rather than blocking the redirect — analytics are lossy
// under extreme backpressure, redirects never are.
type Kafka struct {
	ch      chan ClickEvent
	w       *kafka.Writer
	metrics *observability.Metrics
	done    chan struct{}
}

// NewKafka builds the emitter and starts its background writer goroutine.
func NewKafka(brokers []string, topic string, bufferSize int, m *observability.Metrics) *Kafka {
	k := &Kafka{
		ch: make(chan ClickEvent, bufferSize),
		w: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{}, // key by code → ordered per link, spread across partitions
			BatchTimeout: 200 * time.Millisecond,
			BatchSize:    1000,
			RequiredAcks: kafka.RequireOne,
			Async:        false,
		},
		metrics: m,
		done:    make(chan struct{}),
	}
	go k.run()
	return k
}

// Emit enqueues without blocking. On a full buffer the event is dropped.
func (k *Kafka) Emit(e ClickEvent) {
	select {
	case k.ch <- e:
		if k.metrics != nil {
			k.metrics.EventsEmitted.Inc()
		}
	default:
		if k.metrics != nil {
			k.metrics.EventsDropped.Inc()
		}
	}
}

func (k *Kafka) run() {
	defer close(k.done)
	const maxBatch = 1000
	batch := make([]kafka.Message, 0, maxBatch)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = k.w.WriteMessages(ctx, batch...) // best-effort; loss is acceptable per design
		cancel()
		batch = batch[:0]
	}

	for {
		select {
		case e, ok := <-k.ch:
			if !ok {
				flush()
				return
			}
			if data, err := json.Marshal(e); err == nil {
				batch = append(batch, kafka.Message{Key: []byte(e.Code), Value: data})
			}
			if len(batch) >= maxBatch {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// Close drains the buffer and closes the writer, bounded by ctx.
func (k *Kafka) Close(ctx context.Context) error {
	close(k.ch)
	select {
	case <-k.done:
	case <-ctx.Done():
	}
	return k.w.Close()
}

package gojinn

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

func (r *Gojinn) runBackgroundJob(wasmFile string) {
	ctx, span := otel.Tracer("gojinn-scheduler").Start(context.Background(), "cron_trigger")
	defer span.End()

	cronPayload := `{"event_type": "cron", "source": "gojinn_scheduler"}`

	r.runAsyncJob(ctx, wasmFile, cronPayload)
}

func (r *Gojinn) runAsyncJob(ctx context.Context, wasmFile, payload string) {
	tracer := otel.Tracer("gojinn-publisher")
	ctx, span := tracer.Start(ctx, "publish_async_job")
	defer span.End()

	if r.js == nil {
		err := fmt.Errorf("JetStream not ready")
		r.logger.Error("Cannot queue async job", zap.String("file", wasmFile), zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	topic := fmt.Sprintf("gojinn.exec.%s", hashString(wasmFile))

	jobPayload := struct {
		Method  string              `json:"method"`
		URI     string              `json:"uri"`
		Headers map[string][]string `json:"headers"`
		Body    string              `json:"body"`
	}{
		Method: "ASYNC",
		URI:    "internal://async/job",
		Headers: map[string][]string{
			"X-Source": {"internal"},
		},
		Body: payload,
	}

	jobBytes, err := json.Marshal(jobPayload)
	if err != nil {
		r.logger.Error("Failed to marshal async job", zap.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "json marshal failed")
		return
	}

	msgID := fmt.Sprintf("job_%d", time.Now().UnixNano())

	msg := nats.NewMsg(topic)
	msg.Data = jobBytes

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(msg.Header))

	_, err = r.js.PublishMsg(msg, nats.MsgId(msgID))
	if err != nil {
		r.logger.Error("Failed to persist async job",
			zap.String("file", wasmFile),
			zap.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "nats publish failed")
		return
	}

	r.logger.Info("Async Job Persisted & Queued",
		zap.String("file", wasmFile),
		zap.String("msg_id", msgID),
		zap.String("trace_id", span.SpanContext().TraceID().String()),
	)
}

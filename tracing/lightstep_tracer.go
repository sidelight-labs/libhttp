package tracing

import (
	"context"
	"github.com/lightstep/otel-launcher-go/launcher"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Tracer interface {
	Trace(commandName string) Span
	Shutdown()
}

type Span interface {
	End()
}

type LightstepTracer struct {
	launcher launcher.Launcher
	tracer   trace.Tracer
	ctx      context.Context
	spanID   string
}

func NewLightstepTracer(service, token string) *LightstepTracer {
	return &LightstepTracer{
		launcher: launcher.ConfigureOpentelemetry(
			launcher.WithServiceName(service),
			launcher.WithAccessToken(token),
		),
		tracer: otel.Tracer(service),
	}
}

func (t *LightstepTracer) Trace(name string) Span {
	if t.ctx == nil {
		t.ctx = context.Background()
	}

	var span trace.Span

	if t.spanID == "" {
		t.ctx, span = t.tracer.Start(t.ctx, name)
		t.spanID = span.SpanContext().SpanID().String()
	} else {
		_, span = t.tracer.Start(t.ctx, name)
	}
	
	return &LightstepSpan{span: span}
}

func (t *LightstepTracer) Shutdown() {
	t.launcher.Shutdown()
}

type LightstepSpan struct {
	span trace.Span
}

func (s *LightstepSpan) End() {
	if s.span != nil {
		s.span.End()
	}
}

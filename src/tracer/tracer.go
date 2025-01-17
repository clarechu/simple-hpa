package tracer

import (
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"io"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/uber/jaeger-client-go/config"
)

// New returns a new tracer
func New(serviceName, hostPort string) (opentracing.Tracer, io.Closer) {
	cfg := config.Configuration{
		ServiceName: serviceName,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  hostPort, // localhost:5775
		},
	}

	tracer, closer, err := cfg.NewTracer(
		config.Logger(jaeger.NullLogger),
	)

	if err != nil {
		log.Error().Err(err).Msg("fail creating tracer")
	}

	return tracer, closer
}

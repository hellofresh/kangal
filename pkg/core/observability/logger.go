package observability

import (
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggerConfig is the possible logger configurations
type LoggerConfig struct {
	Level string `envconfig:"LOG_LEVEL" default:"info"`
	Type  string `envconfig:"LOG_TYPE" default:"kangal"`
}

type stdLoggerWriter struct {
	logger *zap.Logger
}

func (w *stdLoggerWriter) Write(p []byte) (int, error) {
	w.logger.Info(string(p), zap.String("source", "std-log"))
	return len(p), nil
}

// NewLogger initializes and returns logger instance and flag if it is in debug mode
func NewLogger(cfg LoggerConfig) (*zap.Logger, bool, error) {
	var err error
	logConfig := zap.NewProductionConfig()

	logLevel := new(zap.AtomicLevel)
	if err := logLevel.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, false, err
	}

	logConfig.Level = *logLevel
	logConfig.Development = logLevel.String() == zapcore.DebugLevel.String()
	logConfig.Sampling = nil
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logConfig.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	logConfig.InitialFields = map[string]interface{}{"type": cfg.Type}

	logger, err := logConfig.Build()
	if err != nil {
		return nil, false, err
	}

	// override std logger to write into app logger
	log.SetOutput(&stdLoggerWriter{logger})

	return logger, logConfig.Development, nil
}

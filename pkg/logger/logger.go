package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New() (*zap.Logger, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	encoder := zapcore.NewJSONEncoder(encoderConfig)

	stdout := zapcore.AddSync(os.Stdout)

	file, err := os.OpenFile("/logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	fileWriter := zapcore.AddSync(file)

	level := zapcore.InfoLevel

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, stdout, level),
		zapcore.NewCore(encoder, fileWriter, level),
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}

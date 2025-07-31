package zapLogger

import (
	"io"
	"os"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	once sync.Once
	Log  *zap.SugaredLogger
)

// Init initializes zap logger and returns the opened log file handle
func Init() *os.File {
	var logFile *os.File
	once.Do(func() {
		// Ensure logs directory exists (optional)
		// os.MkdirAll("logs", os.ModePerm)

		var err error
		logFile, err = os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic("cannot open log file: " + err.Error())
		}

		fileWriter := zapcore.AddSync(logFile)
		consoleWriter := zapcore.AddSync(os.Stdout)

		encoderCfg := zap.NewProductionEncoderConfig()
		encoderCfg.TimeKey = "timestamp"
		encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderCfg),
			zapcore.NewMultiWriteSyncer(consoleWriter, fileWriter),
			zap.InfoLevel,
		)

		logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
		Log = logger.Sugar()
	})
	return logFile
}

// FiberLoggingMiddleware returns Fiber's built-in logger middleware writing logs to stdout and given logFile
func FiberLoggingMiddleware(logFile *os.File) fiber.Handler {
	return logger.New(logger.Config{
		Output: io.MultiWriter(os.Stdout, logFile),
		// Format: "${time} | ${status} | ${method} | ${path} | ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone: "Local",
	})
}

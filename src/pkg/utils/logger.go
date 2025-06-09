package utils

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

type Config struct {
	LogLevel  string `mapstructure:"level"`  // debug, info, warn, error
	LogFormat string `mapstructure:"format"` // json, text
	Pretty    bool   `mapstructure:"pretty"` // Pour JSON pretty print
}

// Logger étend logrus.Logger pour ajouter des fonctionnalités personnalisées
type Logger struct {
	*logrus.Logger
	config Config
}

// NewLogger crée une nouvelle instance de logger avec la config fournie
func NewLogger(config Config) *Logger {
	log := &Logger{
		Logger: logrus.New(),
		config: config,
	}

	// Configuration du format
	switch config.LogFormat {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			PrettyPrint: config.Pretty,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := filepath.Base(f.File)
				return f.Function, filename + ":" + string(rune(f.Line))
			},
		})
	default:
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,  // Force les couleurs même en dehors d'un TTY
			DisableColors: false, // Activer les couleurs
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := filepath.Base(f.File)
				return f.Function, filename + ":" + string(rune(f.Line))
			},
		})
	}

	// Configuration du niveau de log
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// Activation du caller
	log.SetReportCaller(true)
	log.SetOutput(os.Stdout)

	return log
}

// Helper functions pour les champs communs
func (l *Logger) WithFunc() *logrus.Entry {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return l.WithField("caller", "unknown")
	}

	fn := runtime.FuncForPC(pc)
	return l.WithFields(logrus.Fields{
		"func": fn.Name(),
		"file": filepath.Base(file),
		"line": line,
	})
}

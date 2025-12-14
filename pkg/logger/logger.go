package logger

import (
	"io"
	"log"
	"os"
)

// Level representa o nível de log
type Level int

const (
	DEBUG Level = iota
	INFO
	WARNING
	ERROR
)

// Logger é nossa estrutura de logging
type Logger struct {
	infoLogger    *log.Logger
	warningLogger *log.Logger
	errorLogger   *log.Logger
	debugLogger   *log.Logger
	minLevel      Level
}

// New cria um novo logger
func New(output io.Writer, minLevel Level) *Logger {
	flags := log.Ldate | log.Ltime | log.Lmicroseconds

	return &Logger{
		infoLogger:    log.New(output, "INFO:    ", flags),
		warningLogger: log.New(output, "WARNING: ", flags),
		errorLogger:   log.New(os.Stderr, "ERROR:   ", flags),
		debugLogger:   log.New(output, "DEBUG:   ", flags),
		minLevel:      minLevel,
	}
}

// Default cria um logger padrão para stdout
func Default() *Logger {
	return New(os.Stdout, INFO)
}

// Info loga mensagens informativas
func (l *Logger) Info(msg string) {
	if l.minLevel <= INFO {
		l.infoLogger.Println(msg)
	}
}

// Infof loga mensagens informativas com formatação
func (l *Logger) Infof(format string, v ...interface{}) {
	if l.minLevel <= INFO {
		l.infoLogger.Printf(format, v...)
	}
}

// Warning loga avisos
func (l *Logger) Warning(msg string) {
	if l.minLevel <= WARNING {
		l.warningLogger.Println(msg)
	}
}

// Warningf loga avisos com formatação
func (l *Logger) Warningf(format string, v ...interface{}) {
	if l.minLevel <= WARNING {
		l.warningLogger.Printf(format, v...)
	}
}

// Error loga erros
func (l *Logger) Error(msg string) {
	if l.minLevel <= ERROR {
		l.errorLogger.Println(msg)
	}
}

// Errorf loga erros com formatação
func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.minLevel <= ERROR {
		l.errorLogger.Printf(format, v...)
	}
}

// Debug loga mensagens de debug
func (l *Logger) Debug(msg string) {
	if l.minLevel <= DEBUG {
		l.debugLogger.Println(msg)
	}
}

// Debugf loga mensagens de debug com formatação
func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.minLevel <= DEBUG {
		l.debugLogger.Printf(format, v...)
	}
}

// Global logger instance
var defaultLogger = Default()

// Package-level functions para usar o logger global

// Info loga uma mensagem informativa
func Info(msg string) {
	defaultLogger.Info(msg)
}

// Infof loga uma mensagem informativa com formatação
func Infof(format string, v ...interface{}) {
	defaultLogger.Infof(format, v...)
}

// Warning loga um aviso
func Warning(msg string) {
	defaultLogger.Warning(msg)
}

// Warningf loga um aviso com formatação
func Warningf(format string, v ...interface{}) {
	defaultLogger.Warningf(format, v...)
}

// Error loga um erro
func Error(msg string) {
	defaultLogger.Error(msg)
}

// Errorf loga um erro com formatação
func Errorf(format string, v ...interface{}) {
	defaultLogger.Errorf(format, v...)
}

// Debug loga uma mensagem de debug
func Debug(msg string) {
	defaultLogger.Debug(msg)
}

// Debugf loga uma mensagem de debug com formatação
func Debugf(format string, v ...interface{}) {
	defaultLogger.Debugf(format, v...)
}

// SetLevel define o nível mínimo de log do logger global
func SetLevel(level Level) {
	defaultLogger.minLevel = level
}

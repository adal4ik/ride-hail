package main

import "log"

type Logger struct{}

func (l *Logger) Info(msg string, args ...interface{}) {
	log.Printf(Green+"[INFO] "+Reset+msg, args...)
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	log.Printf(Yellow+"[WARN] "+Reset+msg, args...)
}

func (l *Logger) Error(msg string, args ...interface{}) {
	log.Printf(Red+"[ERROR] "+Reset+msg, args...)
}

func (l *Logger) WebSocket(msg string, args ...interface{}) {
	log.Printf(Cyan+"[WS] "+Reset+msg, args...)
}

func (l *Logger) HTTP(msg string, args ...interface{}) {
	log.Printf(Gray+"[HTTP] "+Reset+msg, args...)
}

package main

type progressReporter interface {
	Add(num int) error
}

type progressReporterFactory func(max int64, description ...string) progressReporter

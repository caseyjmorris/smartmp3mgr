package main

import "github.com/schollz/progressbar/v3"

type progressReporter interface {
	Add(num int) error
}

type progressReporterFactory func(max int64, description ...string) progressReporter

func pr() progressReporter {
	return progressbar.Default(2)
}

func pr2() progressReporterFactory {
	return func(max int64, description ...string) progressReporter {
		var pr progressReporter
		pr = progressbar.Default(max, description...)
		return pr
	}
}

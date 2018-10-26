// Copyright 2018 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package loggers

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"github.com/gohugoio/hugo/common/terminal"

	jww "github.com/spf13/jwalterweatherman"
)

var (
	// Counts ERROR logs to the global jww logger.
	GlobalErrorCounter *jww.Counter
)

func init() {
	GlobalErrorCounter = &jww.Counter{}
	jww.SetLogListeners(jww.LogCounter(GlobalErrorCounter, jww.LevelError))
}

// Logger wraps a *loggers.Logger and some other related logging state.
type Logger struct {
	*jww.Notepad
	ErrorCounter *jww.Counter

	// This is only set in server mode.
	errors *bytes.Buffer
}

func (l *Logger) Errors() string {
	if l.errors == nil {
		return ""
	}
	return ansiColorRe.ReplaceAllString(l.errors.String(), "")
}

// Reset resets the logger's internal state.
func (l *Logger) Reset() {
	l.ErrorCounter.Reset()
	if l.errors != nil {
		l.errors.Reset()
	}
}

//  NewLogger creates a new Logger for the given thresholds
func NewLogger(stdoutThreshold, logThreshold jww.Threshold, outHandle, logHandle io.Writer, saveErrors bool) *Logger {
	return newLogger(stdoutThreshold, logThreshold, outHandle, logHandle, saveErrors)
}

// NewDebugLogger is a convenience function to create a debug logger.
func NewDebugLogger() *Logger {
	return newBasicLogger(jww.LevelDebug)
}

// NewWarningLogger is a convenience function to create a warning logger.
func NewWarningLogger() *Logger {
	return newBasicLogger(jww.LevelWarn)
}

// NewErrorLogger is a convenience function to create an error logger.
func NewErrorLogger() *Logger {
	return newBasicLogger(jww.LevelError)
}

var ansiColorRe = regexp.MustCompile("(?s)\\033\\[\\d*(;\\d*)*m")

type ansiCleaner struct {
	w io.Writer
}

func (a ansiCleaner) Write(p []byte) (n int, err error) {
	return a.w.Write(ansiColorRe.ReplaceAll(p, []byte("")))
}

func newLogger(stdoutThreshold, logThreshold jww.Threshold, outHandle, logHandle io.Writer, saveErrors bool) *Logger {
	errorCounter := &jww.Counter{}
	if logHandle != ioutil.Discard && terminal.IsTerminal(os.Stdout) {
		// Remove any Ansi coloring from log output
		logHandle = ansiCleaner{w: logHandle}
	}
	listeners := []jww.LogListener{jww.LogCounter(errorCounter, jww.LevelError)}
	var errorBuff *bytes.Buffer
	if saveErrors {
		errorBuff = new(bytes.Buffer)
		errorCapture := func(t jww.Threshold) io.Writer {
			if t != jww.LevelError {
				// Only interested in ERROR
				return nil
			}

			return errorBuff
		}

		listeners = append(listeners, errorCapture)
	}

	return &Logger{
		Notepad:      jww.NewNotepad(stdoutThreshold, logThreshold, outHandle, logHandle, "", log.Ldate|log.Ltime, listeners...),
		ErrorCounter: errorCounter,
		errors:       errorBuff,
	}
}

func newBasicLogger(t jww.Threshold) *Logger {
	return newLogger(t, jww.LevelError, os.Stdout, ioutil.Discard, false)
}

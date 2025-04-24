package assert

import (
	"log/slog"
	"reflect"
	"runtime"
	"runtime/debug"
)

const SENTINEL = "assertion failure"

func runAssert(msg string, args ...any) {
	debug.PrintStack()

	// Grab the frame of the function raising the panic
	pc := make([]uintptr, 4)    // Collect up to 4 stack frames
	n := runtime.Callers(3, pc) // Skip 3 frames: runtime.Callers, runAssert and the assert function parent
	frames := runtime.CallersFrames(pc[:n])
	callingFunctionFrame, _ := frames.Next()

	slogValues := []any{
		"msg",
		msg,
		"file",
		callingFunctionFrame.File,
		"line",
		callingFunctionFrame.Line,
		"function",
		callingFunctionFrame.Function,
	}
	slogValues = append(slogValues, args...)

	slog.Error("ASSERT")
	for i := 0; i < len(slogValues); i += 2 {
		slog.Error(">", slogValues[i], slogValues[i+1])
	}
	panic(SENTINEL)
}

// Assert that the given value evaluates to true
func Assert(truth bool, msg string, data ...any) {
	if !truth {
		runAssert(msg, data...)
	}
}

// Assert that the given item is not nil
func NotNil(item any, msg string, data ...any) {
	if item == nil {
		slog.Error("NotNil#nil encountered")
		runAssert(msg, data...)
	}

	if reflect.ValueOf(item).Kind() == reflect.Ptr && reflect.ValueOf(item).IsNil() {
		slog.Error("NotNil#nil encountered")
		runAssert(msg, data...)
	}
}

func NoError(err error, msg string, data ...any) {
	if err != nil {
		slog.Error("Error#NotNil encountered")
		data = append(data, "err")
		data = append(data, err)
		runAssert(msg, data...)
	}

	if reflect.ValueOf(err).Kind() == reflect.Ptr && !reflect.ValueOf(err).IsNil() {
		slog.Error("Nil#NotNil encountered")
		data = append(data, "err")
		data = append(data, err)
		runAssert(msg, data...)
	}
}

func Nil(item any, msg string, data ...any) {
	if item != nil {
		slog.Error("Nil#Nil encountered")
		runAssert(msg, data...)
	}

	if reflect.ValueOf(item).Kind() == reflect.Ptr && !reflect.ValueOf(item).IsNil() {
		slog.Error("Nil#Nil encountered")
		runAssert(msg, data...)
	}
}


package object

import (
	"bytes"
	"fmt"
	"slug/internal/util"
)

func RenderStacktrace(rtErr *RuntimeError) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "RuntimeError: %s\n\n", rtErr.Payload.Inspect())

	if len(rtErr.StackTrace) > 0 {
		l, c := util.GetLineAndColumn(rtErr.StackTrace[0].Src, rtErr.StackTrace[0].Position)
		buf.WriteString(util.GetContextLines(rtErr.StackTrace[0].Src, l, c))
		buf.WriteString("\n")
	}

	// Start with the payload itself
	fmt.Fprintf(&buf, "Stack trace: %s", rtErr.Payload.Inspect())
	buf.WriteString(formatRuntimeErrorStack(rtErr))

	return buf.String()
}

// Helper: turn a RuntimeError's stack trace into a human-readable string.
func formatRuntimeErrorStack(rtErr *RuntimeError) string {
	var buf bytes.Buffer

	for _, frame := range rtErr.StackTrace {
		l, c := util.GetLineAndColumn(frame.Src, frame.Position)
		fmt.Fprintf(&buf, "\n  at [%3d:%3d] %-8s - %s", l, c, frame.Function, frame.File)
	}

	// Optionally include chained causes
	if rtErr.Cause != nil {
		fmt.Fprintf(&buf, "\nCaused by: %s", rtErr.Cause.Payload.Inspect())
		buf.WriteString(formatRuntimeErrorStack(rtErr.Cause))
	}

	return buf.String()
}

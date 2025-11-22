package util

import (
	"bytes"
	"fmt"
)

func GetLineAndColumn(src string, pos int) (line int, column int) {
	line = 1
	column = 1
	for i, char := range src {
		if i == pos {
			break
		}
		if char == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return
}

// GetContextLines extracts and formats context lines around an error position
func GetContextLines(src string, errorLine, errorCol, errorPos int) string {
	var result bytes.Buffer

	// Split source into lines
	lines := []string{}
	currentLine := 1
	lineStart := 0

	for i, ch := range src {
		if ch == '\n' || i == len(src)-1 {
			if i == len(src)-1 && ch != '\n' {
				lines = append(lines, src[lineStart:i+1])
			} else {
				lines = append(lines, src[lineStart:i])
			}
			lineStart = i + 1
			currentLine++
		}
	}

	// Show 2 lines before the error line (if available)
	startLine := errorLine - 2
	if startLine < 1 {
		startLine = 1
	}

	// Format and write context lines
	for i := startLine; i <= errorLine && i <= len(lines); i++ {
		lineNum := i
		lineContent := ""
		if i <= len(lines) {
			lineContent = lines[i-1]
		}

		if i == errorLine {
			// Error line with arrow
			margin := fmt.Sprintf("  >  %3d | ", lineNum)
			result.WriteString(fmt.Sprintf("%s%s\n", margin, lineContent))
			result.WriteString(fmt.Sprintf("%s^ unexpected here",
				replaceVisibleWithSpaces(margin+lineContent[:errorCol-1])))
		} else {
			// Context line
			result.WriteString(fmt.Sprintf("     %3d | %s\n", lineNum, lineContent))
		}
	}

	return result.String()
}

// replaceVisibleWithSpaces replaces all non-whitespace characters with spaces
// while preserving tabs for correct alignment.
func replaceVisibleWithSpaces(s string) string {
	var buf bytes.Buffer
	for _, c := range s {
		if c == '\t' {
			buf.WriteRune('\t')
		} else {
			buf.WriteRune(' ')
		}
	}
	return buf.String()
}

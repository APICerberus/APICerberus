package yaml

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

type token struct {
	line     int
	indent   int
	original string
	cleaned  string
}

func (t token) isSkippable() bool {
	return strings.TrimSpace(t.cleaned) == ""
}

func (t token) isBlankLine() bool {
	return strings.TrimSpace(t.original) == ""
}

func tokenize(data []byte) ([]token, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	tokens := make([]token, 0)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimRight(scanner.Text(), "\r")

		indent, err := leadingSpaces(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}

		content := line[indent:]
		cleaned := strings.TrimSpace(stripInlineComment(content))
		tokens = append(tokens, token{
			line:     lineNo,
			indent:   indent,
			original: line,
			cleaned:  cleaned,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return tokens, nil
}

func leadingSpaces(s string) (int, error) {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ':
			continue
		case '\t':
			return 0, fmt.Errorf("tab indentation is not supported")
		default:
			return i, nil
		}
	}
	return len(s), nil
}

func stripInlineComment(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			b.WriteByte(ch)
			escaped = false
			continue
		}

		if inDouble && ch == '\\' {
			b.WriteByte(ch)
			escaped = true
			continue
		}

		if !inDouble && ch == '\'' {
			inSingle = !inSingle
			b.WriteByte(ch)
			continue
		}
		if !inSingle && ch == '"' {
			inDouble = !inDouble
			b.WriteByte(ch)
			continue
		}

		if !inSingle && !inDouble && ch == '#' {
			break
		}

		b.WriteByte(ch)
	}

	return strings.TrimRight(b.String(), " \t")
}

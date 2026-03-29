package yaml

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse parses YAML bytes into an internal node tree.
func Parse(data []byte) (Node, error) {
	tokens, err := tokenize(data)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return newNodeMap(), nil
	}

	p := &parser{tokens: tokens}
	node, err := p.parseDocument()
	if err != nil {
		return nil, err
	}

	p.skipSkippable()
	if p.pos < len(p.tokens) {
		t := p.tokens[p.pos]
		return nil, fmt.Errorf("line %d: unexpected content: %q", t.line, t.cleaned)
	}
	return node, nil
}

type parser struct {
	tokens []token
	pos    int
}

func (p *parser) parseDocument() (Node, error) {
	p.skipSkippable()
	if p.done() {
		return newNodeMap(), nil
	}

	t := p.peek()
	return p.parseBlock(t.indent)
}

func (p *parser) parseBlock(indent int) (Node, error) {
	p.skipSkippable()
	if p.done() {
		return &NodeScalar{Value: ""}, nil
	}

	t := p.peek()
	if t.indent < indent {
		return &NodeScalar{Value: ""}, nil
	}
	if t.indent > indent {
		return nil, fmt.Errorf("line %d: unexpected indentation: got %d, expected %d", t.line, t.indent, indent)
	}

	if isSequenceItem(t.cleaned) {
		return p.parseSequence(indent)
	}
	return p.parseMap(indent)
}

func (p *parser) parseMap(indent int) (Node, error) {
	out := newNodeMap()

	for {
		p.skipSkippable()
		if p.done() {
			break
		}

		t := p.peek()
		if t.indent < indent {
			break
		}
		if t.indent > indent {
			return nil, fmt.Errorf("line %d: unexpected indentation in map", t.line)
		}
		if isSequenceItem(t.cleaned) {
			return nil, fmt.Errorf("line %d: sequence item not allowed in map context", t.line)
		}

		key, rawValue, ok := splitMapEntry(t.cleaned)
		if !ok {
			return nil, fmt.Errorf("line %d: invalid map entry: %q", t.line, t.cleaned)
		}
		keyValue, err := parseScalarToken(key)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid key: %w", t.line, err)
		}

		p.pos++
		value, err := p.parseValue(indent, rawValue)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", t.line, err)
		}
		out.set(keyValue, value)
	}

	return out, nil
}

func (p *parser) parseSequence(indent int) (Node, error) {
	out := &NodeSequence{Items: make([]Node, 0)}

	for {
		p.skipSkippable()
		if p.done() {
			break
		}

		t := p.peek()
		if t.indent < indent {
			break
		}
		if t.indent > indent {
			return nil, fmt.Errorf("line %d: unexpected indentation in sequence", t.line)
		}
		if !isSequenceItem(t.cleaned) {
			break
		}

		item := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(t.cleaned), "-"))
		p.pos++

		itemNode, err := p.parseSequenceItem(indent, item)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", t.line, err)
		}
		out.Items = append(out.Items, itemNode)
	}

	return out, nil
}

func (p *parser) parseSequenceItem(sequenceIndent int, item string) (Node, error) {
	switch item {
	case "":
		return p.parseNestedOrEmpty(sequenceIndent)
	case "|":
		return p.parseMultiline(sequenceIndent, true), nil
	case ">":
		return p.parseMultiline(sequenceIndent, false), nil
	}

	if key, value, ok := splitMapEntry(item); ok {
		mapIndent := sequenceIndent + 2
		out := newNodeMap()

		keyValue, err := parseScalarToken(key)
		if err != nil {
			return nil, fmt.Errorf("invalid sequence map key: %w", err)
		}

		firstValue, err := p.parseValue(mapIndent, value)
		if err != nil {
			return nil, err
		}
		out.set(keyValue, firstValue)

		for {
			snapshot := p.pos
			p.skipSkippable()
			if p.done() {
				break
			}

			t := p.peek()
			if t.indent < mapIndent {
				break
			}
			if t.indent > mapIndent {
				return nil, fmt.Errorf("line %d: unexpected indentation in sequence map item", t.line)
			}

			if isSequenceItem(t.cleaned) {
				p.pos = snapshot
				break
			}

			k, v, ok := splitMapEntry(t.cleaned)
			if !ok {
				break
			}

			keyName, err := parseScalarToken(k)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid map key: %w", t.line, err)
			}

			p.pos++
			valNode, err := p.parseValue(mapIndent, v)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", t.line, err)
			}
			out.set(keyName, valNode)
		}

		return out, nil
	}

	value, err := parseScalarToken(item)
	if err != nil {
		return nil, err
	}
	return &NodeScalar{Value: value}, nil
}

func (p *parser) parseValue(parentIndent int, raw string) (Node, error) {
	value := strings.TrimSpace(raw)
	switch value {
	case "":
		return p.parseNestedOrEmpty(parentIndent)
	case "|":
		return p.parseMultiline(parentIndent, true), nil
	case ">":
		return p.parseMultiline(parentIndent, false), nil
	default:
		scalar, err := parseScalarToken(value)
		if err != nil {
			return nil, err
		}
		return &NodeScalar{Value: scalar}, nil
	}
}

func (p *parser) parseNestedOrEmpty(parentIndent int) (Node, error) {
	snapshot := p.pos
	p.skipSkippable()
	if p.done() {
		return &NodeScalar{Value: ""}, nil
	}

	t := p.peek()
	if t.indent <= parentIndent {
		p.pos = snapshot
		return &NodeScalar{Value: ""}, nil
	}
	return p.parseBlock(t.indent)
}

func (p *parser) parseMultiline(parentIndent int, literal bool) Node {
	working := p.pos
	lines := make([]string, 0)
	blockIndent := -1

	for working < len(p.tokens) {
		t := p.tokens[working]
		if strings.TrimSpace(t.original) == "" {
			if blockIndent == -1 {
				working++
				continue
			}
			lines = append(lines, "")
			working++
			continue
		}
		if t.indent <= parentIndent {
			break
		}
		if blockIndent == -1 {
			blockIndent = t.indent
		}

		content := trimNLeadingSpaces(t.original, blockIndent)
		lines = append(lines, content)
		working++
	}

	p.pos = working
	if blockIndent == -1 {
		return &NodeScalar{Value: ""}
	}

	if literal {
		return &NodeScalar{Value: strings.Join(lines, "\n")}
	}
	return &NodeScalar{Value: foldLines(lines)}
}

func (p *parser) skipSkippable() {
	for p.pos < len(p.tokens) && p.tokens[p.pos].isSkippable() {
		p.pos++
	}
}

func (p *parser) done() bool {
	return p.pos >= len(p.tokens)
}

func (p *parser) peek() token {
	return p.tokens[p.pos]
}

func isSequenceItem(s string) bool {
	trimmed := strings.TrimSpace(s)
	return trimmed == "-" || strings.HasPrefix(trimmed, "- ")
}

func splitMapEntry(s string) (key, value string, ok bool) {
	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if inDouble && ch == '\\' {
			escaped = true
			continue
		}
		if !inDouble && ch == '\'' {
			inSingle = !inSingle
			continue
		}
		if !inSingle && ch == '"' {
			inDouble = !inDouble
			continue
		}
		if inSingle || inDouble {
			continue
		}

		if ch == ':' {
			if i+1 < len(s) && s[i+1] != ' ' {
				continue
			}
			key = strings.TrimSpace(s[:i])
			value = strings.TrimSpace(s[i+1:])
			return key, value, key != ""
		}
	}

	return "", "", false
}

func parseScalarToken(s string) (string, error) {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) < 2 {
		return trimmed, nil
	}

	if trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'' {
		inner := trimmed[1 : len(trimmed)-1]
		return strings.ReplaceAll(inner, "''", "'"), nil
	}
	if trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
		unquoted, err := strconv.Unquote(trimmed)
		if err != nil {
			return "", err
		}
		return unquoted, nil
	}
	return trimmed, nil
}

func trimNLeadingSpaces(s string, n int) string {
	if n <= 0 {
		return s
	}
	i := 0
	for i < len(s) && i < n && s[i] == ' ' {
		i++
	}
	return s[i:]
}

func foldLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(lines[0])
	for i := 1; i < len(lines); i++ {
		prev := lines[i-1]
		curr := lines[i]

		if prev == "" || curr == "" {
			b.WriteByte('\n')
		} else {
			b.WriteByte(' ')
		}
		b.WriteString(curr)
	}
	return b.String()
}

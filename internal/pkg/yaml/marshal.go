package yaml

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Marshal encodes a Go value to YAML.
func Marshal(v any) ([]byte, error) {
	var b strings.Builder
	if err := writeYAMLValue(&b, reflect.ValueOf(v), 0); err != nil {
		return nil, err
	}

	out := b.String()
	if out == "" {
		return []byte{}, nil
	}
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return []byte(out), nil
}

func writeYAMLValue(b *strings.Builder, value reflect.Value, indent int) error {
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			writeIndent(b, indent)
			b.WriteString("null\n")
			return nil
		}
		value = value.Elem()
	}

	if !value.IsValid() {
		writeIndent(b, indent)
		b.WriteString("null\n")
		return nil
	}

	switch value.Kind() {
	case reflect.Struct:
		return writeStruct(b, value, indent)
	case reflect.Map:
		return writeMap(b, value, indent)
	case reflect.Slice, reflect.Array:
		return writeSequence(b, value, indent)
	default:
		writeIndent(b, indent)
		scalar, err := scalarToYAML(value)
		if err != nil {
			return err
		}
		b.WriteString(scalar)
		b.WriteByte('\n')
		return nil
	}
}

func writeStruct(b *strings.Builder, value reflect.Value, indent int) error {
	wrote := false
	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		if field.PkgPath != "" {
			continue
		}

		name, skip := yamlFieldName(field)
		if skip {
			continue
		}

		if err := writeMapEntry(b, name, value.Field(i), indent); err != nil {
			return err
		}
		wrote = true
	}
	if !wrote {
		writeIndent(b, indent)
		b.WriteString("{}\n")
	}
	return nil
}

func writeMap(b *strings.Builder, value reflect.Value, indent int) error {
	if value.IsNil() || value.Len() == 0 {
		writeIndent(b, indent)
		b.WriteString("{}\n")
		return nil
	}

	keys := value.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%v", keys[i].Interface()) < fmt.Sprintf("%v", keys[j].Interface())
	})

	for _, key := range keys {
		entryName := fmt.Sprintf("%v", key.Interface())
		if err := writeMapEntry(b, entryName, value.MapIndex(key), indent); err != nil {
			return err
		}
	}
	return nil
}

func writeMapEntry(b *strings.Builder, key string, value reflect.Value, indent int) error {
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			writeIndent(b, indent)
			b.WriteString(key)
			b.WriteString(": null\n")
			return nil
		}
		value = value.Elem()
	}

	if !value.IsValid() {
		writeIndent(b, indent)
		b.WriteString(key)
		b.WriteString(": null\n")
		return nil
	}

	if value.Kind() == reflect.String && strings.Contains(value.String(), "\n") {
		writeIndent(b, indent)
		b.WriteString(key)
		b.WriteString(": |\n")
		for _, line := range strings.Split(value.String(), "\n") {
			writeIndent(b, indent+2)
			b.WriteString(line)
			b.WriteByte('\n')
		}
		return nil
	}

	if isInlineScalar(value) {
		scalar, err := scalarToYAML(value)
		if err != nil {
			return err
		}
		writeIndent(b, indent)
		b.WriteString(key)
		b.WriteString(": ")
		b.WriteString(scalar)
		b.WriteByte('\n')
		return nil
	}

	writeIndent(b, indent)
	b.WriteString(key)
	b.WriteString(":\n")
	return writeYAMLValue(b, value, indent+2)
}

func writeSequence(b *strings.Builder, value reflect.Value, indent int) error {
	if value.Kind() == reflect.Slice && value.IsNil() {
		writeIndent(b, indent)
		b.WriteString("[]\n")
		return nil
	}
	if value.Len() == 0 {
		writeIndent(b, indent)
		b.WriteString("[]\n")
		return nil
	}

	for i := 0; i < value.Len(); i++ {
		item := value.Index(i)
		for item.IsValid() && (item.Kind() == reflect.Interface || item.Kind() == reflect.Pointer) {
			if item.IsNil() {
				writeIndent(b, indent)
				b.WriteString("- null\n")
				goto nextItem
			}
			item = item.Elem()
		}

		if !item.IsValid() {
			writeIndent(b, indent)
			b.WriteString("- null\n")
			continue
		}

		if item.Kind() == reflect.String && strings.Contains(item.String(), "\n") {
			writeIndent(b, indent)
			b.WriteString("- |\n")
			for _, line := range strings.Split(item.String(), "\n") {
				writeIndent(b, indent+2)
				b.WriteString(line)
				b.WriteByte('\n')
			}
			continue
		}

		if isInlineScalar(item) {
			scalar, err := scalarToYAML(item)
			if err != nil {
				return err
			}
			writeIndent(b, indent)
			b.WriteString("- ")
			b.WriteString(scalar)
			b.WriteByte('\n')
			continue
		}

		writeIndent(b, indent)
		b.WriteString("-\n")
		if err := writeYAMLValue(b, item, indent+2); err != nil {
			return err
		}

	nextItem:
	}

	return nil
}

func isInlineScalar(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	case reflect.String:
		return !strings.Contains(value.String(), "\n")
	default:
		return value.Type() == durationType || value.Type() == reflect.TypeOf(time.Time{})
	}
}

func scalarToYAML(value reflect.Value) (string, error) {
	if !value.IsValid() {
		return "null", nil
	}

	if value.Type() == durationType {
		d := value.Interface().(time.Duration)
		return quoteIfNeeded(d.String()), nil
	}
	if value.Type() == reflect.TypeOf(time.Time{}) {
		t := value.Interface().(time.Time)
		return quoteIfNeeded(t.Format(time.RFC3339Nano)), nil
	}

	switch value.Kind() {
	case reflect.String:
		return quoteIfNeeded(value.String()), nil
	case reflect.Bool:
		if value.Bool() {
			return "true", nil
		}
		return "false", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(value.Uint(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(value.Float(), 'f', -1, 32), nil
	case reflect.Float64:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("unsupported scalar type: %s", value.Type())
	}
}

func quoteIfNeeded(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, ":\n#{}[],'\"") || strings.HasPrefix(s, "- ") || strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") {
		return strconv.Quote(s)
	}

	lower := strings.ToLower(s)
	switch lower {
	case "true", "false", "null", "yes", "no", "on", "off":
		return strconv.Quote(s)
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return strconv.Quote(s)
	}
	return s
}

func writeIndent(b *strings.Builder, n int) {
	for i := 0; i < n; i++ {
		b.WriteByte(' ')
	}
}

package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func applyEnvOverrides(cfg *Config) error {
	if cfg == nil {
		return nil
	}
	return applyEnvToStruct("APICERBERUS", reflect.ValueOf(cfg).Elem())
}

func applyEnvToStruct(prefix string, value reflect.Value) error {
	t := value.Type()
	for i := 0; i < value.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}

		field := value.Field(i)
		envName := prefix + "_" + envSegment(sf)

		if isNestedStructField(field) {
			if err := applyEnvToStruct(envName, field); err != nil {
				return err
			}
			continue
		}

		raw, ok := os.LookupEnv(envName)
		if !ok {
			continue
		}
		if err := setFromString(field, raw); err != nil {
			return fmt.Errorf("%s: %w", envName, err)
		}
	}
	return nil
}

func envSegment(field reflect.StructField) string {
	tag := field.Tag.Get("yaml")
	if tag != "" {
		name := strings.TrimSpace(strings.Split(tag, ",")[0])
		if name != "" && name != "-" {
			return strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
		}
	}
	return strings.ToUpper(toSnakeCase(field.Name))
}

func isNestedStructField(field reflect.Value) bool {
	if field.Kind() != reflect.Struct {
		return false
	}
	return field.Type() != reflect.TypeOf(time.Time{})
}

func setFromString(field reflect.Value, raw string) error {
	if !field.CanSet() {
		return fmt.Errorf("field cannot be set")
	}

	if field.Type() == reflect.TypeOf(time.Duration(0)) {
		d, err := time.ParseDuration(strings.TrimSpace(raw))
		if err != nil {
			return err
		}
		field.SetInt(int64(d))
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
		return nil
	case reflect.Bool:
		v, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return err
		}
		field.SetBool(v)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(v)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v, err := strconv.ParseUint(strings.TrimSpace(raw), 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(v)
		return nil
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(strings.TrimSpace(raw), field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(v)
		return nil
	default:
		return fmt.Errorf("unsupported env override type: %s", field.Type())
	}
}

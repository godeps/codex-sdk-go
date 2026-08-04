package codex

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const internalOriginatorEnv = "CODEX_INTERNAL_ORIGINATOR_OVERRIDE"
const goSDKOriginator = "codex_sdk_go"

func flattenConfigOverrides(config map[string]any) ([]string, error) {
	if len(config) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(config))
	for key := range config {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	overrides := make([]string, 0, len(config))
	for _, key := range keys {
		flattened, err := flattenConfigValue(key, reflect.ValueOf(config[key]))
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, flattened...)
	}
	return overrides, nil
}

func flattenConfigValue(path string, value reflect.Value) ([]string, error) {
	value = unwrapReflectValue(value)
	if !value.IsValid() {
		return nil, fmt.Errorf("config %s: nil values are not supported", path)
	}

	if value.Kind() == reflect.Map {
		if value.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("config %s: map keys must be strings", path)
		}

		keys := make([]string, 0, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			keys = append(keys, iter.Key().String())
		}
		sort.Strings(keys)

		overrides := make([]string, 0, value.Len())
		for _, key := range keys {
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			flattened, err := flattenConfigValue(childPath, value.MapIndex(reflect.ValueOf(key)))
			if err != nil {
				return nil, err
			}
			overrides = append(overrides, flattened...)
		}
		return overrides, nil
	}

	literal, err := encodeTOMLLiteral(value)
	if err != nil {
		return nil, fmt.Errorf("config %s: %w", path, err)
	}
	return []string{path + "=" + literal}, nil
}

func encodeTOMLLiteral(value reflect.Value) (string, error) {
	value = unwrapReflectValue(value)
	if !value.IsValid() {
		return "", errors.New("nil values are not supported")
	}

	if number, ok := value.Interface().(json.Number); ok {
		return number.String(), nil
	}

	switch value.Kind() {
	case reflect.String:
		encoded, err := json.Marshal(value.String())
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	case reflect.Bool:
		return strconv.FormatBool(value.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(value.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return formatTOMLFloat(value.Float()), nil
	case reflect.Slice, reflect.Array:
		parts := make([]string, 0, value.Len())
		for i := range value.Len() {
			part, err := encodeTOMLLiteral(value.Index(i))
			if err != nil {
				return "", err
			}
			parts = append(parts, part)
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return "", errors.New("map keys must be strings")
		}

		keys := make([]string, 0, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			keys = append(keys, iter.Key().String())
		}
		sort.Strings(keys)

		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			part, err := encodeTOMLLiteral(value.MapIndex(reflect.ValueOf(key)))
			if err != nil {
				return "", err
			}
			parts = append(parts, key+"="+part)
		}
		return "{" + strings.Join(parts, ", ") + "}", nil
	default:
		return "", fmt.Errorf("unsupported value type %s", value.Kind())
	}
}

func unwrapReflectValue(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func formatTOMLFloat(value float64) string {
	switch {
	case math.IsNaN(value):
		return "nan"
	case math.IsInf(value, 1):
		return "inf"
	case math.IsInf(value, -1):
		return "-inf"
	default:
		return strconv.FormatFloat(value, 'g', -1, 64)
	}
}

func buildEnv(override map[string]string, baseURL string, apiKey string) []string {
	env := map[string]string{}
	if override != nil {
		for key, value := range override {
			env[key] = value
		}
	} else {
		for _, entry := range os.Environ() {
			parts := strings.SplitN(entry, "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}
	}
	if _, ok := env[internalOriginatorEnv]; !ok {
		env[internalOriginatorEnv] = goSDKOriginator
	}
	if baseURL != "" {
		env["OPENAI_BASE_URL"] = baseURL
	}
	if apiKey != "" {
		env["CODEX_API_KEY"] = apiKey
	}
	list := make([]string, 0, len(env))
	for key, value := range env {
		list = append(list, key+"="+value)
	}
	return list
}

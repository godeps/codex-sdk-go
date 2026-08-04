package codex

import "reflect"

func isJSONObject(value any) bool {
	if value == nil {
		return false
	}
	switch reflect.ValueOf(value).Kind() {
	case reflect.Map, reflect.Struct:
		return true
	default:
		return false
	}
}

package gonest

import (
	"reflect"
	"strings"
)

// SerializerInterceptor transforms response objects based on struct tags.
// Use the `serialize` tag to control field visibility:
//   - `serialize:"expose"` — always include this field
//   - `serialize:"exclude"` — always exclude this field
//   - `serialize:"group=admin"` — only include when "admin" group is active
//
// Set groups via route metadata: .SetMetadata("serialize_groups", []string{"admin"})
type SerializerInterceptor struct{}

func NewSerializerInterceptor() *SerializerInterceptor {
	return &SerializerInterceptor{}
}

func (i *SerializerInterceptor) Intercept(ctx ExecutionContext, next CallHandler) (any, error) {
	result, err := next.Handle()
	if err != nil {
		return nil, err
	}

	// Check if handler stored data for serialization via context store.
	// This pattern is used when the handler defers response writing to the interceptor.
	fromStore := false
	if result == nil {
		data, ok := ctx.Get("__serialize_data")
		if ok {
			result = data
			fromStore = true
		}
	}
	if result == nil {
		return nil, nil
	}

	groups, _ := GetMetadata[[]string](ctx, "serialize_groups")
	transformed := serializeValue(reflect.ValueOf(result), groups)

	// If data came from the context store, the handler deferred writing to us.
	if fromStore && !ctx.Written() {
		statusCode := 200
		if code, ok := ctx.Get("__serialize_status"); ok {
			if c, ok := code.(int); ok {
				statusCode = c
			}
		}
		return nil, ctx.JSON(statusCode, transformed)
	}

	return transformed, nil
}

func serializeValue(v reflect.Value, groups []string) any {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Slice {
		result := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = serializeValue(v.Index(i), groups)
		}
		return result
	}

	if v.Kind() != reflect.Struct {
		return v.Interface()
	}

	result := make(map[string]any)
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		serializeTag := field.Tag.Get("serialize")
		jsonTag := field.Tag.Get("json")

		// Determine field name
		name := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				name = parts[0]
			}
			if parts[0] == "-" {
				continue
			}
		}

		// Check serialize tag
		if serializeTag != "" {
			if serializeTag == "exclude" {
				continue
			}
			if strings.HasPrefix(serializeTag, "group=") {
				requiredGroup := serializeTag[6:]
				if !containsGroup(groups, requiredGroup) {
					continue
				}
			}
			// "expose" or unknown tags: include the field
		}

		fieldVal := v.Field(i)
		if fieldVal.Kind() == reflect.Struct || fieldVal.Kind() == reflect.Ptr || fieldVal.Kind() == reflect.Slice {
			result[name] = serializeValue(fieldVal, groups)
		} else {
			result[name] = fieldVal.Interface()
		}
	}

	return result
}

func containsGroup(groups []string, target string) bool {
	for _, g := range groups {
		if g == target {
			return true
		}
	}
	return false
}

package gonest

import (
	"reflect"
	"strings"
)

// ValidationPipe validates request bodies using struct tags.
// Uses the "validate" struct tag with rules: required, min, max, gte, lte, email.
// For production use, integrate with github.com/go-playground/validator/v10.
type ValidationPipe struct{}

// NewValidationPipe creates a new validation pipe.
func NewValidationPipe() *ValidationPipe {
	return &ValidationPipe{}
}

func (p *ValidationPipe) Transform(value any, metadata ArgumentMetadata) (any, error) {
	if metadata.Type != "body" {
		return value, nil
	}
	if value == nil {
		return nil, NewBadRequestException("validation failed: body is required")
	}

	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return value, nil
	}

	t := v.Type()
	var errors []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}

		rules := strings.Split(tag, ",")

		// If the first rule is "omitempty" and the field is zero, skip all rules.
		if len(rules) > 0 && strings.TrimSpace(rules[0]) == "omitempty" {
			if isZero(fieldVal) {
				continue
			}
			rules = rules[1:]
		}

		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if err := validateRule(field.Name, fieldVal, rule); err != "" {
				errors = append(errors, err)
			}
		}
	}

	if len(errors) > 0 {
		return nil, NewBadRequestException("validation failed: " + strings.Join(errors, "; "))
	}

	return value, nil
}

func validateRule(fieldName string, val reflect.Value, rule string) string {
	switch {
	case rule == "required":
		if isZero(val) {
			return fieldName + " is required"
		}
	case strings.HasPrefix(rule, "min="):
		minStr := rule[4:]
		if val.Kind() == reflect.String {
			min := parseInt(minStr)
			if len(val.String()) < min {
				return fieldName + " must be at least " + minStr + " characters"
			}
		}
	case strings.HasPrefix(rule, "max="):
		maxStr := rule[4:]
		if val.Kind() == reflect.String {
			max := parseInt(maxStr)
			if len(val.String()) > max {
				return fieldName + " must be at most " + maxStr + " characters"
			}
		}
	case strings.HasPrefix(rule, "gte="):
		gteStr := rule[4:]
		gte := parseInt(gteStr)
		if val.CanInt() && val.Int() < int64(gte) {
			return fieldName + " must be >= " + gteStr
		}
	case strings.HasPrefix(rule, "lte="):
		lteStr := rule[4:]
		lte := parseInt(lteStr)
		if val.CanInt() && val.Int() > int64(lte) {
			return fieldName + " must be <= " + lteStr
		}
	case rule == "email":
		if val.Kind() == reflect.String {
			s := val.String()
			if !strings.Contains(s, "@") || !strings.Contains(s, ".") {
				return fieldName + " must be a valid email"
			}
		}
	case strings.HasPrefix(rule, "oneof="):
		allowed := strings.Fields(rule[6:])
		s := ""
		if val.Kind() == reflect.String {
			s = val.String()
		} else {
			s = strings.TrimSpace(val.String())
		}
		found := false
		for _, a := range allowed {
			if s == a {
				found = true
				break
			}
		}
		if !found {
			return fieldName + " must be one of: " + strings.Join(allowed, ", ")
		}
	}
	return ""
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map:
		return v.IsNil()
	default:
		return false
	}
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

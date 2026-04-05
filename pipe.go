package gonest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ArgumentMetadata describes the argument being processed by a pipe.
type ArgumentMetadata struct {
	Type string // "param", "query", "body", "header", "custom"
	Name string // parameter name
}

// Pipe transforms or validates a value before it reaches the handler.
// Equivalent to NestJS PipeTransform.
type Pipe interface {
	Transform(value any, metadata ArgumentMetadata) (any, error)
}

// PipeFunc is a convenience adapter for simple pipe functions.
type PipeFunc func(value any, metadata ArgumentMetadata) (any, error)

func (f PipeFunc) Transform(value any, metadata ArgumentMetadata) (any, error) {
	return f(value, metadata)
}

// ParseIntPipe transforms a string parameter to an int.
type ParseIntPipe struct {
	ParamName string
}

func NewParseIntPipe(paramName string) *ParseIntPipe {
	return &ParseIntPipe{ParamName: paramName}
}

func (p *ParseIntPipe) Transform(value any, metadata ArgumentMetadata) (any, error) {
	if metadata.Name != p.ParamName && p.ParamName != "" {
		return value, nil
	}
	str, ok := value.(string)
	if !ok {
		return nil, NewBadRequestException("Validation failed: expected string for " + metadata.Name)
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return nil, NewBadRequestException("Validation failed: " + metadata.Name + " must be an integer")
	}
	return val, nil
}

// ParseBoolPipe transforms a string parameter to a bool.
type ParseBoolPipe struct {
	ParamName string
}

func NewParseBoolPipe(paramName string) *ParseBoolPipe {
	return &ParseBoolPipe{ParamName: paramName}
}

func (p *ParseBoolPipe) Transform(value any, metadata ArgumentMetadata) (any, error) {
	if metadata.Name != p.ParamName && p.ParamName != "" {
		return value, nil
	}
	str, ok := value.(string)
	if !ok {
		return nil, NewBadRequestException("Validation failed: expected string for " + metadata.Name)
	}
	val, err := strconv.ParseBool(str)
	if err != nil {
		return nil, NewBadRequestException("Validation failed: " + metadata.Name + " must be a boolean")
	}
	return val, nil
}

// ParseFloatPipe transforms a string parameter to a float64.
type ParseFloatPipe struct {
	ParamName string
}

func NewParseFloatPipe(paramName string) *ParseFloatPipe {
	return &ParseFloatPipe{ParamName: paramName}
}

func (p *ParseFloatPipe) Transform(value any, metadata ArgumentMetadata) (any, error) {
	if metadata.Name != p.ParamName && p.ParamName != "" {
		return value, nil
	}
	str, ok := value.(string)
	if !ok {
		return nil, NewBadRequestException("Validation failed: expected string for " + metadata.Name)
	}
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return nil, NewBadRequestException("Validation failed: " + metadata.Name + " must be a number")
	}
	return val, nil
}

// ParseUUIDPipe validates that a string parameter is a valid UUID.
type ParseUUIDPipe struct {
	ParamName string
}

func NewParseUUIDPipe(paramName string) *ParseUUIDPipe {
	return &ParseUUIDPipe{ParamName: paramName}
}

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (p *ParseUUIDPipe) Transform(value any, metadata ArgumentMetadata) (any, error) {
	if metadata.Name != p.ParamName && p.ParamName != "" {
		return value, nil
	}
	str, ok := value.(string)
	if !ok {
		return nil, NewBadRequestException("Validation failed: expected string for " + metadata.Name)
	}
	if !uuidRegex.MatchString(str) {
		return nil, NewBadRequestException("Validation failed: " + metadata.Name + " must be a UUID")
	}
	return str, nil
}

// DefaultValuePipe provides a default value when the input is nil or empty string.
type DefaultValuePipe struct {
	ParamName    string
	DefaultValue any
}

func NewDefaultValuePipe(paramName string, defaultValue any) *DefaultValuePipe {
	return &DefaultValuePipe{ParamName: paramName, DefaultValue: defaultValue}
}

func (p *DefaultValuePipe) Transform(value any, metadata ArgumentMetadata) (any, error) {
	if metadata.Name != p.ParamName && p.ParamName != "" {
		return value, nil
	}
	if value == nil {
		return p.DefaultValue, nil
	}
	if str, ok := value.(string); ok && str == "" {
		return p.DefaultValue, nil
	}
	return value, nil
}

// ParseArrayPipe splits a comma-separated string into a slice of strings.
type ParseArrayPipe struct {
	ParamName string
	Separator string
}

func NewParseArrayPipe(paramName string) *ParseArrayPipe {
	return &ParseArrayPipe{ParamName: paramName, Separator: ","}
}

func (p *ParseArrayPipe) Transform(value any, metadata ArgumentMetadata) (any, error) {
	if metadata.Name != p.ParamName && p.ParamName != "" {
		return value, nil
	}
	str, ok := value.(string)
	if !ok {
		return nil, NewBadRequestException("Validation failed: expected string for " + metadata.Name)
	}
	if str == "" {
		return []string{}, nil
	}
	sep := p.Separator
	if sep == "" {
		sep = ","
	}
	result := splitAndTrim(str, sep)
	return result, nil
}

func splitAndTrim(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			part := trimSpace(s[start:i])
			if part != "" {
				result = append(result, part)
			}
			start = i + len(sep)
		}
	}
	part := trimSpace(s[start:])
	if part != "" {
		result = append(result, part)
	}
	return result
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// ParseDatePipe transforms a string parameter to a time.Time.
// Supports RFC3339, date-only (2006-01-02), and custom formats.
type ParseDatePipe struct {
	ParamName string
	Format    string // optional custom format; defaults to RFC3339 then date-only
}

func NewParseDatePipe(paramName string) *ParseDatePipe {
	return &ParseDatePipe{ParamName: paramName}
}

func (p *ParseDatePipe) Transform(value any, metadata ArgumentMetadata) (any, error) {
	if metadata.Name != p.ParamName && p.ParamName != "" {
		return value, nil
	}
	str, ok := value.(string)
	if !ok {
		return nil, NewBadRequestException("Validation failed: expected string for " + metadata.Name)
	}

	if p.Format != "" {
		t, err := time.Parse(p.Format, str)
		if err != nil {
			return nil, NewBadRequestException(
				fmt.Sprintf("Validation failed: %s must be a valid date (%s)", metadata.Name, p.Format))
		}
		return t, nil
	}

	// Try RFC3339 first, then date-only
	if t, err := time.Parse(time.RFC3339, str); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", str); err == nil {
		return t, nil
	}
	return nil, NewBadRequestException(
		"Validation failed: " + metadata.Name + " must be a valid date (RFC3339 or YYYY-MM-DD)")
}

// ParseEnumPipe validates that a string parameter is one of the allowed enum values.
type ParseEnumPipe struct {
	ParamName string
	Values    []string
}

func NewParseEnumPipe(paramName string, values ...string) *ParseEnumPipe {
	return &ParseEnumPipe{ParamName: paramName, Values: values}
}

func (p *ParseEnumPipe) Transform(value any, metadata ArgumentMetadata) (any, error) {
	if metadata.Name != p.ParamName && p.ParamName != "" {
		return value, nil
	}
	str, ok := value.(string)
	if !ok {
		return nil, NewBadRequestException("Validation failed: expected string for " + metadata.Name)
	}
	for _, v := range p.Values {
		if str == v {
			return str, nil
		}
	}
	return nil, NewBadRequestException(
		fmt.Sprintf("Validation failed: %s must be one of [%s]", metadata.Name, strings.Join(p.Values, ", ")))
}

// ParseFilePipe validates uploaded files using a chain of FileValidator rules.
// Use ParseFilePipeBuilder for a fluent construction API.
type ParseFilePipe struct {
	FieldName  string
	Validators []FileValidator
}

// FileValidator validates an uploaded file.
type FileValidator interface {
	Validate(file *UploadedFile) error
}

func NewParseFilePipe(fieldName string, validators ...FileValidator) *ParseFilePipe {
	return &ParseFilePipe{FieldName: fieldName, Validators: validators}
}

func (p *ParseFilePipe) Transform(value any, metadata ArgumentMetadata) (any, error) {
	file, ok := value.(*UploadedFile)
	if !ok {
		return nil, NewBadRequestException("Validation failed: expected file for " + p.FieldName)
	}
	for _, v := range p.Validators {
		if err := v.Validate(file); err != nil {
			return nil, err
		}
	}
	return file, nil
}

// ParseFilePipeBuilder provides a fluent API for building a ParseFilePipe.
type ParseFilePipeBuilder struct {
	fieldName  string
	validators []FileValidator
}

func NewParseFilePipeBuilder(fieldName string) *ParseFilePipeBuilder {
	return &ParseFilePipeBuilder{fieldName: fieldName}
}

func (b *ParseFilePipeBuilder) AddFileTypeValidator(allowedTypes ...string) *ParseFilePipeBuilder {
	b.validators = append(b.validators, &FileTypeValidator{AllowedTypes: allowedTypes})
	return b
}

func (b *ParseFilePipeBuilder) AddMaxSizeValidator(maxSize int64) *ParseFilePipeBuilder {
	b.validators = append(b.validators, &FileSizeValidator{MaxSize: maxSize})
	return b
}

func (b *ParseFilePipeBuilder) Build() *ParseFilePipe {
	return NewParseFilePipe(b.fieldName, b.validators...)
}

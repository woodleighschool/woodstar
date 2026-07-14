// Package validation provides the shared structural validation boundary.
package validation

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	playground "github.com/go-playground/validator/v10"
)

var structValidator = newStructValidator()

// Struct validates value using its validate tags.
func Struct(value any) error {
	err := structValidator.Struct(value)
	if err == nil {
		return nil
	}

	var validationErrors playground.ValidationErrors
	ok := errors.As(err, &validationErrors)
	if !ok || len(validationErrors) == 0 {
		return err
	}
	return fieldError{validationErrors[0]}
}

// IsHTTPSOrigin reports whether value is an HTTPS origin without a path,
// credentials, query, or fragment.
func IsHTTPSOrigin(value string) bool {
	return isWebOrigin(value, true)
}

func newStructValidator() *playground.Validate {
	validate := playground.New(playground.WithRequiredStructEnabled())
	if err := validate.RegisterValidation("https_origin", func(field playground.FieldLevel) bool {
		return IsHTTPSOrigin(field.Field().String())
	}); err != nil {
		panic(err)
	}
	if err := validate.RegisterValidation("https_url", func(field playground.FieldLevel) bool {
		parsed, err := url.Parse(field.Field().String())
		return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil
	}); err != nil {
		panic(err)
	}
	if err := validate.RegisterValidation("web_origin", func(field playground.FieldLevel) bool {
		return isWebOrigin(field.Field().String(), false)
	}); err != nil {
		panic(err)
	}
	if err := validate.RegisterValidation("notblank", func(field playground.FieldLevel) bool {
		return strings.TrimSpace(field.Field().String()) != ""
	}); err != nil {
		panic(err)
	}
	return validate
}

func isWebOrigin(value string, httpsOnly bool) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" ||
		parsed.ForceQuery || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return false
	}
	if httpsOnly {
		return parsed.Scheme == "https"
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

type fieldError struct {
	playground.FieldError
}

func (err fieldError) Error() string {
	field := err.Field()
	switch err.Tag() {
	case "required", "required_if", "required_unless", "required_with", "required_with_all":
		return field + " is required"
	case "notblank":
		return field + " must not be blank"
	case "excluded_if",
		"excluded_unless",
		"excluded_with",
		"excluded_with_all",
		"excluded_without",
		"excluded_without_all":
		return field + " is not allowed"
	case "min", "gte":
		return fmt.Sprintf("%s must be at least %s", field, err.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, err.Param())
	case "max", "lte":
		return fmt.Sprintf("%s must be at most %s", field, err.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, err.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of %s", field, err.Param())
	case "cidr":
		return field + " must be a CIDR"
	case "https_origin":
		return field + " must be an HTTPS origin"
	case "https_url":
		return field + " must be an HTTPS URL"
	case "web_origin":
		return field + " must be an HTTP or HTTPS origin"
	case "unique":
		return field + " must not contain duplicates"
	default:
		return field + " is invalid"
	}
}

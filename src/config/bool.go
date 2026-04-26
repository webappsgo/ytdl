// Package config handles configuration loading and parsing helpers.
package config

import (
	"fmt"
	"strings"
)

var truthyValues = map[string]bool{
	"1": true, "y": true, "t": true,
	"yes": true, "true": true, "on": true, "ok": true,
	"enable": true, "enabled": true,
	"yep": true, "yup": true, "yeah": true,
	"aye": true, "si": true, "oui": true, "da": true, "hai": true,
	"affirmative": true, "accept": true, "allow": true, "grant": true,
	"sure": true, "totally": true,
}

var falsyValues = map[string]bool{
	"0": true, "n": true, "f": true,
	"no": true, "false": true, "off": true,
	"disable": true, "disabled": true,
	"nope": true, "nah": true, "nay": true,
	"nein": true, "non": true, "niet": true, "iie": true, "lie": true,
	"negative": true, "reject": true, "block": true, "revoke": true,
	"deny": true, "never": true, "noway": true,
}

// ParseBool parses a string into a boolean using the repository truthy/falsy set.
// Empty values return the provided default.
func ParseBool(value string, defaultValue bool) (bool, error) {
	normalizedValue := strings.TrimSpace(strings.ToLower(value))

	if normalizedValue == "" {
		return defaultValue, nil
	}

	if truthyValues[normalizedValue] {
		return true, nil
	}

	if falsyValues[normalizedValue] {
		return false, nil
	}

	return false, fmt.Errorf("invalid boolean value: %q", value)
}

// MustParseBool parses a string into a boolean and panics on invalid values.
func MustParseBool(value string, defaultValue bool) bool {
	parsedValue, err := ParseBool(value, defaultValue)
	if err != nil {
		panic(err)
	}

	return parsedValue
}

// IsTruthy returns true only for known truthy values.
func IsTruthy(value string) bool {
	return truthyValues[strings.TrimSpace(strings.ToLower(value))]
}

// IsFalsy returns true only for known falsy values.
func IsFalsy(value string) bool {
	return falsyValues[strings.TrimSpace(strings.ToLower(value))]
}

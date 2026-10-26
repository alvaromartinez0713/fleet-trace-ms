// Package config provides typed helpers to read environment variables.
//
// It depends only on the standard library and has no knowledge of specific
// keys. Each service maps its own keys into its config struct in its own
// composition layer.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrRequired is returned, wrapped with the key name, when a required variable is missing or empty.
	ErrRequired = errors.New("required environment variable not set")

	// ErrInvalid is returned, wrapped with the key name, when a variable cannot be parsed.
	ErrInvalid = errors.New("invalid environment variable value")
)

// Env reads typed values from a lookup source and collects problems.
// Read all values first, then call Err to get every problem at once.
// Error messages name the key only. They never include the value, since values may be secrets.
type Env struct {
	lookup func(string) (string, bool)
	errs   []error
}

// NewEnv returns an Env backed by lookup. It is useful for tests or for values
// that do not come from the process environment.
func NewEnv(lookup func(string) (string, bool)) *Env {
	return &Env{lookup: lookup}
}

// OSEnv returns an Env backed by the process environment.
func OSEnv() *Env {
	return NewEnv(os.LookupEnv)
}

// get returns the trimmed value for key. An empty value counts as unset.
func (e *Env) get(key string) (string, bool) {
	v, ok := e.lookup(key)
	v = strings.TrimSpace(v)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

// String returns the value for key, or def when the variable is unset.
func (e *Env) String(key, def string) string {
	if v, ok := e.get(key); ok {
		return v
	}
	return def
}

// Required returns the value for key. It records ErrRequired when the variable is unset.
func (e *Env) Required(key string) string {
	v, ok := e.get(key)
	if !ok {
		e.errs = append(e.errs, fmt.Errorf("%s: %w", key, ErrRequired))
		return ""
	}
	return v
}

// Int returns the value for key parsed as an int, or def when the variable is unset.
func (e *Env) Int(key string, def int) int {
	v, ok := e.get(key)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s: %w: expected an integer", key, ErrInvalid))
		return def
	}
	return n
}

// Bool returns the value for key parsed with strconv.ParseBool, or def when the variable is unset.
func (e *Env) Bool(key string, def bool) bool {
	v, ok := e.get(key)
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s: %w: expected true or false", key, ErrInvalid))
		return def
	}
	return b
}

// Duration returns the value for key parsed with time.ParseDuration, or def when the variable is unset.
func (e *Env) Duration(key string, def time.Duration) time.Duration {
	v, ok := e.get(key)
	if !ok {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s: %w: expected a duration such as 10s", key, ErrInvalid))
		return def
	}
	return d
}

// StringList returns the value for key split on commas. Items are trimmed and empty items are dropped.
// It returns def when the variable is unset or contains no items.
func (e *Env) StringList(key string, def []string) []string {
	v, ok := e.get(key)
	if !ok {
		return def
	}
	var items []string
	for _, part := range strings.Split(v, ",") {
		if p := strings.TrimSpace(part); p != "" {
			items = append(items, p)
		}
	}
	if len(items) == 0 {
		return def
	}
	return items
}

// Err returns all problems recorded by the helpers above, or nil when there are none.
func (e *Env) Err() error {
	return errors.Join(e.errs...)
}

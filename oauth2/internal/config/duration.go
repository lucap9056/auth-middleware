package config

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidDuration = errors.New("invalid duration")

var durationUnits = map[byte]time.Duration{
	'd': 24 * time.Hour,
	'h': time.Hour,
	'm': time.Minute,
	's': time.Second,
}

func parseDuration(value string, bareUnit time.Duration) (time.Duration, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return 0, fmt.Errorf("%w: empty value", ErrInvalidDuration)
	}

	if isDigits(normalized) {
		if bareUnit == 0 {
			return 0, fmt.Errorf("%w: %q is missing a unit (d, h, m, s)", ErrInvalidDuration, value)
		}
		return multiplyDuration(normalized, bareUnit, value)
	}

	var total time.Duration
	rest := normalized
	for rest != "" {
		digitsEnd := 0
		for digitsEnd < len(rest) && isDigit(rest[digitsEnd]) {
			digitsEnd++
		}
		if digitsEnd == 0 || digitsEnd == len(rest) {
			return 0, fmt.Errorf("%w: %q must be a sequence of <integer><unit>", ErrInvalidDuration, value)
		}

		unit, ok := durationUnits[rest[digitsEnd]]
		if !ok {
			return 0, fmt.Errorf("%w: %q has unknown unit %q", ErrInvalidDuration, value, rest[digitsEnd])
		}

		part, err := multiplyDuration(rest[:digitsEnd], unit, value)
		if err != nil {
			return 0, err
		}
		if total > math.MaxInt64-part {
			return 0, fmt.Errorf("%w: %q is out of range", ErrInvalidDuration, value)
		}
		total += part
		rest = rest[digitsEnd+1:]
	}
	return total, nil
}

func multiplyDuration(digits string, unit time.Duration, original string) (time.Duration, error) {
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || n > math.MaxInt64/int64(unit) {
		return 0, fmt.Errorf("%w: %q is out of range", ErrInvalidDuration, original)
	}
	return time.Duration(n) * unit, nil
}

func isDigits(s string) bool {
	for i := range len(s) {
		if !isDigit(s[i]) {
			return false
		}
	}
	return true
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

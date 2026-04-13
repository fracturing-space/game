package canonical

import (
	"strings"
	"unicode/utf8"

	"github.com/fracturing-space/game/internal/errs"
)

// DisplayNameMaxRunes is the default maximum length for user-facing display names.
const DisplayNameMaxRunes = 128

// Errorf matches the formatting helpers used by boundary validators.
type Errorf func(string, ...any) error

// IsExact reports whether value has no surrounding whitespace.
func IsExact(value string) bool {
	return value == strings.TrimSpace(value)
}

// IsBlank reports whether value is empty after trimming whitespace.
func IsBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}

// ValidateName reports whether one normalized user-facing name is present and
// within the supplied rune limit.
func ValidateName(value, label string, maxRunes int) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errs.InvalidArgumentf("%s is required", label)
	}
	if maxRunes > 0 && utf8.RuneCountInString(value) > maxRunes {
		return errs.InvalidArgumentf("%s must be %d characters or fewer", label, maxRunes)
	}
	return nil
}

// ValidateExact reports whether one required exact token-style value is present.
func ValidateExact(value, label string, errorf Errorf) error {
	if value == "" {
		return errorf("%s is required", label)
	}
	if !IsExact(value) {
		return errorf("%s must not contain surrounding whitespace", label)
	}
	return nil
}

// ValidateOptionalExact reports whether one optional exact token-style value is exact when present.
func ValidateOptionalExact(value, label string, errorf Errorf) error {
	if value != "" && !IsExact(value) {
		return errorf("%s must not contain surrounding whitespace", label)
	}
	return nil
}

// ValidateRelativePath reports whether one required relative path-like value is present and canonical.
func ValidateRelativePath(value, label string, errorf Errorf) error {
	if err := ValidateExact(value, label, errorf); err != nil {
		return err
	}
	if len(value) != 0 && value[0] == '/' {
		return errorf("%s must not start with /", label)
	}
	return nil
}

// ValidateID reports whether one required identifier is present and exact.
func ValidateID(value, label string) error {
	return ValidateExact(value, label, errs.InvalidArgumentf)
}

// ValidateOptionalID reports whether one optional identifier is exact when present.
func ValidateOptionalID(value, label string) error {
	return ValidateOptionalExact(value, label, errs.InvalidArgumentf)
}

// IsSystemType reports whether a type uses the reserved sys.<system>.* namespace.
func IsSystemType(value string) bool {
	return strings.HasPrefix(value, "sys.")
}

// HasSystemTypePrefix reports whether value uses the exact sys.<systemID>.* namespace.
func HasSystemTypePrefix(value, systemID string) bool {
	return strings.HasPrefix(value, SystemTypePrefix(systemID))
}

// SystemTypePrefix returns the reserved prefix for one system-owned type.
func SystemTypePrefix(systemID string) string {
	return "sys." + systemID + "."
}

// ValidateOwnedType reports whether one command or event type satisfies core or system namespace rules.
func ValidateOwnedType(kind, typ, systemID string, systemOwned bool, errorf Errorf) error {
	if typ == "" {
		return errorf("%s type is required", kind)
	}
	if !IsExact(typ) {
		return errorf("%s type must not contain surrounding whitespace: %s", kind, typ)
	}
	if !systemOwned {
		if IsSystemType(typ) {
			return errorf("core %s %s must not use sys.* namespace", kind, typ)
		}
		return nil
	}
	if systemID == "" {
		return errorf("system %s %s system id is required", kind, typ)
	}
	if !IsExact(systemID) {
		return errorf("system %s %s system id must not contain surrounding whitespace", kind, typ)
	}
	prefix := SystemTypePrefix(systemID)
	if !HasSystemTypePrefix(typ, systemID) {
		return errorf("system %s %s must use %s* namespace", kind, typ, prefix)
	}
	return nil
}

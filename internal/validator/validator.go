package validator

import "regexp"

var (
	// EmailRX taken from https://html.spec.whatwg.org/#valid-e-mail-address
	EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

// Validator contains a map of validation errors
type Validator struct {
	Errors map[string]string
}

// New is a helper which creates a new empty Validator
func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

func (v *Validator) AddErr(key, msg string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = msg
	}
}

func (v *Validator) Check(ok bool, key, msg string) {
	if !ok {
		v.AddErr(key, msg)
	}
}

// In returns true if a specific value is in the list
func In(value string, list ...string) bool {
	for i := range list {
		if list[i] == value {
			return true
		}
	}
	return false
}

// Matches returns true if a string value match a
// specific regexp patter.
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// Unique returns true if all string values in a slice are unique.
func Unique(values []string) bool {
	seen := make(map[string]struct{})
	for _, v := range values {
		if _, exists := seen[v]; exists {
			return false
		}
		seen[v] = struct{}{}
	}
	return true
}

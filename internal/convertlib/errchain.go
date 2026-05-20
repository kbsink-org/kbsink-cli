package convertlib

import (
	"errors"
	"strings"
)

// ErrChainString joins messages from an error unwrap chain for user-visible logs.
// kbsink's [core.CodedError] only prints code + label, hiding the wrapped root cause;
// WASM and CLI surfaces use this so users see e.g. host transport / TLS details.
// Redundant segments are dropped (e.g. when a longer message already contains a shorter unwrap).
func ErrChainString(err error) string {
	if err == nil {
		return ""
	}
	var parts []string
	for cur := err; cur != nil; cur = errors.Unwrap(cur) {
		msg := strings.TrimSpace(cur.Error())
		if msg == "" {
			continue
		}
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if msg == last || strings.Contains(last, msg) {
				continue
			}
			if strings.Contains(msg, last) {
				parts[len(parts)-1] = msg
				continue
			}
		}
		parts = append(parts, msg)
	}
	if len(parts) == 0 {
		return err.Error()
	}
	return strings.Join(parts, " | ")
}

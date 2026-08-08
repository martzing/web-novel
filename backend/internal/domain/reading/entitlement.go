package reading

import "github.com/mokchan/webnovel-backend/internal/domain/roles"

// Decide reports whether a chapter body must be withheld from the viewer.
//
// It is the single entitlement rule in the system and is deliberately pure, so
// every branch is unit-testable without a database:
//
//	free chapter                 -> readable
//	already unlocked             -> readable
//	the chapter's own translator -> readable
//	administrator                -> readable
//	otherwise                    -> locked
func Decide(priceCoins int, v Viewer, unlocked, isTranslator bool) bool {
	if priceCoins <= 0 {
		return false
	}
	if v.IsAnonymous() {
		return true
	}
	if unlocked || isTranslator || v.HasRole(roles.Admin) {
		return false
	}
	return true
}

// IsTranslatorOf reports whether the viewer is the chapter's assigned
// translator.
func IsTranslatorOf(c Chapter, v Viewer) bool {
	return c.TranslatorID != nil && !v.IsAnonymous() && *c.TranslatorID == v.UserID
}

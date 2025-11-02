package verify

import (
	"net"
	"regexp"
	"strings"
)

// EmailVerifier provides email address verification
type EmailVerifier struct {
	allowList    []string
	blockList    []string
	checkMX      bool
	checkFormat  bool
}

// NewEmailVerifier creates a new email verifier
func NewEmailVerifier() *EmailVerifier {
	return &EmailVerifier{
		checkFormat: true,
		checkMX:     false,
	}
}

// SetAllowList sets allowed email domains
func (v *EmailVerifier) SetAllowList(domains []string) {
	v.allowList = domains
}

// SetBlockList sets blocked email domains
func (v *EmailVerifier) SetBlockList(domains []string) {
	v.blockList = domains
}

// SetCheckMX enables/disables MX record checking
func (v *EmailVerifier) SetCheckMX(check bool) {
	v.checkMX = check
}

// SetCheckFormat enables/disables format checking
func (v *EmailVerifier) SetCheckFormat(check bool) {
	v.checkFormat = check
}

// VerifyEmail verifies an email address
func (v *EmailVerifier) VerifyEmail(email string) bool {
	// Clean the email
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}

	// Format check
	if v.checkFormat {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(email) {
			return false
		}
	}

	// Extract domain
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	domain := strings.ToLower(parts[1])

	// Check block list
	for _, blocked := range v.blockList {
		blockedLower := strings.ToLower(blocked)
		if domain == blockedLower || strings.HasSuffix(domain, "."+blockedLower) {
			return false
		}
	}

	// Check allow list if set
	if len(v.allowList) > 0 {
		allowed := false
		for _, allowedDomain := range v.allowList {
			allowedLower := strings.ToLower(allowedDomain)
			if domain == allowedLower || strings.HasSuffix(domain, "."+allowedLower) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	// Check MX records if enabled
	if v.checkMX {
		mxRecords, err := net.LookupMX(domain)
		if err != nil || len(mxRecords) == 0 {
			return false
		}
	}

	return true
}

// VerifyEmailFormat verifies basic email format without MX checking
func VerifyEmailFormat(email string) bool {
	verifier := NewEmailVerifier()
	return verifier.VerifyEmail(email)
}


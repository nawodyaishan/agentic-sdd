package opencodeadapter

// Strict admission (contract C4): a native prompt may be launched only after
// independently reviewed prerequisites. A launch declaration is never proof of
// consent or of the controls enforcing the declared bounds, so admission also
// requires control evidence that no declaration can supply. While any control
// is unverified, admission stays denied and no owned process is started.

import "strings"

// LaunchDeclaration captures the explicit bounds a caller asserts for a native
// session. Supplying these fields does not grant admission.
type LaunchDeclaration struct {
	Authorization string   `json:"authorization"` // actual authorization reference
	Effects       []string `json:"effects"`       // intended effects
	Model         string   `json:"model"`         // exact model identity
	Data          string   `json:"data"`          // data scope identity
	Deadline      string   `json:"deadline"`      // deadline bound
	Attempts      int      `json:"attempts"`      // attempt bound (>=1)
	TokenBound    int64    `json:"token_bound"`   // token bound (>0)
	SpendBound    string   `json:"spend_bound"`   // spend bound
}

// admissionControls are the independently reviewed controls that no declaration
// can satisfy. Each must be provisioned and evidenced by the user/operator; they
// are listed here so a denial names exactly what remains unmet.
var admissionControls = []string{
	"strict-pre-send-token-proof",
	"credential-isolation",
	"complete-accounting",
	"provider-caps-allowance",
	"auxiliary-retry-prevention",
}

// Admit reports the unmet prerequisites for a native session. Admission is
// granted only when the returned list is empty; under this feature's boundary
// it is always non-empty because the control evidence is not independently
// reviewable from a declaration.
func Admit(d LaunchDeclaration) []string {
	var unmet []string
	if strings.TrimSpace(d.Authorization) == "" {
		unmet = append(unmet, "authorization-reference")
	}
	if len(d.Effects) == 0 {
		unmet = append(unmet, "intended-effects")
	}
	if strings.TrimSpace(d.Model) == "" {
		unmet = append(unmet, "exact-model-identity")
	}
	if strings.TrimSpace(d.Data) == "" {
		unmet = append(unmet, "data-scope-identity")
	}
	if strings.TrimSpace(d.Deadline) == "" {
		unmet = append(unmet, "deadline-bound")
	}
	if d.Attempts < 1 {
		unmet = append(unmet, "attempt-bound")
	}
	if d.TokenBound < 1 {
		unmet = append(unmet, "token-bound")
	}
	if strings.TrimSpace(d.SpendBound) == "" {
		unmet = append(unmet, "spend-bound")
	}
	return append(unmet, admissionControls...)
}

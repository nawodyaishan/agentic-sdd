package opencodeadapter

// Receipt projection (contract C6/T10): compose the deterministic check
// result, the curated run context and an owned-transport native observation
// into one existing v1 receipt, plus the pending-human sidecar when the
// observation stopped on a consequential human question.
//
// Composition is fail-closed: the receipt's checks, selection and execution
// come from the check result exactly (v1 execution is checks-derived); native
// observations contribute only the permitted values (session, model, cost,
// pending); a native turn that did not end in end_turn can never back a
// receipt claiming a completed run; the projection never renders a review
// verdict, so it can never emit Ready. Native-reported usage and cost are
// recorded as observations, never as provider-verified billing.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"agentic-sdd/internal/harnessevidence"
)

// ContextIdentity carries the curated non-code identity digests.
type ContextIdentity struct {
	Config      *string `json:"config"`
	Skills      *string `json:"skills"`
	Environment *string `json:"environment"`
	Runtime     *string `json:"runtime"`
	Harness     *string `json:"harness"`
	Contract    *string `json:"contract"`
}

// ReceiptContext is the user-curated run context for one projection.
type ReceiptContext struct {
	Synthetic        bool                          `json:"synthetic"`
	RunID            string                        `json:"run_id"`
	Mode             string                        `json:"mode"`
	RequestRef       string                        `json:"request_ref"`
	AuthorizationRef *string                       `json:"authorization_ref"`
	Feature          *string                       `json:"feature"`
	Batch            *string                       `json:"batch"`
	TaskIDs          []string                      `json:"task_ids"`
	AllowedEffects   []string                      `json:"allowed_effects"`
	Profile          *string                       `json:"profile"`
	Tools            []string                      `json:"tools"`
	Limits           harnessevidence.Limits        `json:"limits"`
	Identity         ContextIdentity               `json:"identity"`
	IdentityBefore   harnessevidence.ScopeIdentity `json:"identity_before"`
	IdentityAfter    harnessevidence.ScopeIdentity `json:"identity_after"`
	Model            harnessevidence.Model         `json:"model"`
	WorkflowState    *string                       `json:"workflow_state"`
	Findings         []harnessevidence.Finding     `json:"findings"`
	AffectedFiles    []string                      `json:"affected_files"`
	NextAction       string                        `json:"next_action"`
}

// Project builds the v1 receipt. resultBytes are runner result bytes;
// observationBytes are optional owned-transport session outcome bytes;
// pendingDir, when nonempty, is the private directory for a pending sidecar.
// No workspace or canonical state is written; the only write is the optional
// pending sidecar outside active discovery.
func Project(context ReceiptContext, resultBytes, observationBytes []byte, pendingDir string) (harnessevidence.Receipt, string, error) {
	if err := validateContext(context); err != nil {
		return harnessevidence.Receipt{}, "", err
	}
	result, err := harnessevidence.ParseResult(resultBytes)
	if err != nil {
		return harnessevidence.Receipt{}, "", err
	}
	if result.Workspace == "" {
		return harnessevidence.Receipt{}, "", fmt.Errorf("independent check result required")
	}
	if context.IdentityBefore != context.IdentityAfter {
		return harnessevidence.Receipt{}, "", fmt.Errorf("identity changed between check start and end; evidence is not fresh")
	}
	if context.Identity.Contract != nil && *context.Identity.Contract != result.ContractSHA256 {
		return harnessevidence.Receipt{}, "", fmt.Errorf("result/contract identity mismatch")
	}
	observation, err := parseObservation(observationBytes)
	if err != nil {
		return harnessevidence.Receipt{}, "", err
	}
	session, cost, pending, nextAction, err := normalizeObservation(observation, result.Execution, context.NextAction)
	if err != nil {
		return harnessevidence.Receipt{}, "", err
	}
	pendingPath := ""
	if pending != nil && pendingDir != "" {
		pendingPath, err = WritePending(pendingDir, *pending)
		if err != nil {
			return harnessevidence.Receipt{}, "", err
		}
		nextAction = "answer the pending question recorded at " + pendingPath + "; reconcile scope and authority before re-entry"
	}
	checks, err := harnessevidence.ChecksFromResult(result)
	if err != nil {
		return harnessevidence.Receipt{}, "", err
	}
	receipt := harnessevidence.Receipt{
		Version:          1,
		Synthetic:        context.Synthetic,
		RunID:            context.RunID,
		SessionID:        session,
		Mode:             context.Mode,
		Workspace:        result.Workspace,
		RequestRef:       context.RequestRef,
		AuthorizationRef: context.AuthorizationRef,
		Feature:          context.Feature,
		Batch:            context.Batch,
		TaskIDs:          nonNil(context.TaskIDs),
		AllowedEffects:   nonNil(context.AllowedEffects),
		Profile:          context.Profile,
		Tools:            nonNil(context.Tools),
		Limits:           context.Limits,
		Identity: harnessevidence.Identity{
			BaseRevision: &context.IdentityBefore.BaseRevision,
			Diff:         &context.IdentityBefore.Diff,
			Config:       context.Identity.Config,
			Skills:       context.Identity.Skills,
			Environment:  context.Identity.Environment,
			Runtime:      context.Identity.Runtime,
			Harness:      context.Identity.Harness,
			Contract:     context.Identity.Contract,
		},
		Model:         context.Model,
		Selection:     harnessevidence.Selection{Scope: result.Selection.Scope, CheckIDs: nonNil(result.Selection.CheckIDs)},
		Checks:        nonNil(checks),
		Execution:     result.Execution,
		Review:        nil,
		WorkflowState: context.WorkflowState,
		Findings:      nonNil(context.Findings),
		AffectedFiles: nonNil(context.AffectedFiles),
		NextAction:    nextAction,
		CostUSD:       cost,
		Unknowns:      map[string]string{},
	}
	fillUnknowns(&receipt, observation)
	data, err := json.Marshal(receipt)
	if err != nil {
		return harnessevidence.Receipt{}, "", fmt.Errorf("receipt encoding unavailable")
	}
	// Structural gate: the projected receipt must itself parse as strict v1.
	if _, err := harnessevidence.ParseReceipt(data); err != nil {
		return harnessevidence.Receipt{}, "", fmt.Errorf("projected receipt is not valid v1: %w", err)
	}
	return receipt, pendingPath, nil
}

// nonNil keeps v1 array fields as arrays: an omitted optional list is empty,
// never a null that fails the receipt's own structural gate.
func nonNil[T any](list []T) []T {
	if list == nil {
		return []T{}
	}
	return list
}

func validateContext(c ReceiptContext) error {
	if strings.TrimSpace(c.RunID) == "" || strings.TrimSpace(c.RequestRef) == "" || strings.TrimSpace(c.NextAction) == "" {
		return fmt.Errorf("context requires run id, request reference and next action")
	}
	switch c.Mode {
	case "draft", "build", "review", "resume":
		if c.Feature == nil || c.Batch == nil || len(c.TaskIDs) == 0 {
			return fmt.Errorf("feature modes require feature, batch and task identity")
		}
	case "direct-fix":
		if c.Feature != nil || c.Batch != nil || len(c.TaskIDs) != 0 || c.WorkflowState != nil {
			return fmt.Errorf("direct fix has no canonical feature or task state")
		}
	default:
		return fmt.Errorf("unsupported run mode")
	}
	if c.Limits.DeadlineSeconds < 1 || c.Limits.Steps < 1 || c.Limits.TransientRetries < 0 {
		return fmt.Errorf("context limits are out of range")
	}
	if !strings.HasPrefix(c.IdentityBefore.Diff, "sha256-v1:") || !strings.HasPrefix(c.IdentityAfter.Diff, "sha256-v1:") ||
		c.IdentityBefore.BaseRevision == "" || c.IdentityAfter.BaseRevision == "" {
		return fmt.Errorf("context requires measured before/after identities")
	}
	for _, f := range c.Findings {
		if f.ID == "" || f.Location == "" || f.Impact == "" || f.Evidence == "" {
			return fmt.Errorf("context findings require complete fields")
		}
		switch f.Disposition {
		case "valid", "disputed", "needs evidence":
		default:
			return fmt.Errorf("unsupported finding disposition")
		}
	}
	return nil
}

// parseObservation strictly decodes optional owned-transport session bytes.
func parseObservation(data []byte) (*SessionOutcome, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	var outcome SessionOutcome
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&outcome); err != nil {
		return nil, fmt.Errorf("invalid session observation")
	}
	if err := dec.Decode(new(any)); err == nil {
		return nil, fmt.Errorf("invalid session observation")
	}
	if outcome.Execution == "" {
		return nil, nil
	}
	return &outcome, nil
}

// normalizeObservation applies the fail-closed native composition rules and
// returns the receipt session/cost values, the pending summary to persist and
// the receipt next action.
func normalizeObservation(observation *SessionOutcome, resultExecution, contextNextAction string) (*string, *float64, *PendingSummary, string, error) {
	if observation == nil || observation.Execution == "prepared" {
		return nil, nil, nil, contextNextAction, nil
	}
	session := observation.SessionID
	cost := nativeCost(observation)
	switch observation.Execution {
	case "completed":
		if observation.StopReason != "end_turn" {
			return nil, nil, nil, "", fmt.Errorf("native turn end is established only by end_turn")
		}
		return optText(session), cost, nil, contextNextAction, nil
	case "blocked":
		if observation.Pending == nil {
			if resultExecution == "completed" {
				return nil, nil, nil, "", fmt.Errorf("native observation blocked without a pending human question; refusing to receipt a completed run")
			}
			return optText(session), cost, nil, contextNextAction, nil
		}
		return optText(session), cost, observation.Pending, contextNextAction, nil
	case "incomplete", "cancelled":
		if resultExecution == "completed" {
			return nil, nil, nil, "", fmt.Errorf("native turn did not complete; a completed checks result cannot be receipted as a successful run")
		}
		return optText(session), cost, nil, contextNextAction, nil
	default:
		return nil, nil, nil, "", fmt.Errorf("unknown native observation execution")
	}
}

func nativeCost(observation *SessionOutcome) *float64 {
	if observation.Usage == nil || observation.Usage.CostUSD == nil {
		return nil
	}
	value := *observation.Usage.CostUSD
	return &value
}

func optText(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// fillUnknowns records an explicit reason for every null the projection
// creates; a value that is present never needs one.
func fillUnknowns(r *harnessevidence.Receipt, observation *SessionOutcome) {
	note := func(ptr, reason string) {
		r.Unknowns[ptr] = reason
	}
	if r.SessionID == nil {
		if observation == nil || observation.Execution == "prepared" {
			note("/session_id", "no native session observed")
		} else {
			note("/session_id", "native turn reported no session identity")
		}
	}
	if r.CostUSD == nil {
		if observation == nil || observation.Execution == "prepared" {
			note("/cost_usd", "no native cost observation; unknown is not zero")
		} else {
			note("/cost_usd", "native-reported no cost; unknown is not zero")
		}
	}
	for name, value := range map[string]*string{
		"provider": r.Model.Provider, "id": r.Model.ID, "variant": r.Model.Variant,
	} {
		if value == nil {
			note("/model/"+name, "no model identity observed")
		}
	}
	for name, value := range map[string]*string{
		"config": r.Identity.Config, "skills": r.Identity.Skills, "environment": r.Identity.Environment,
		"runtime": r.Identity.Runtime, "harness": r.Identity.Harness, "contract": r.Identity.Contract,
	} {
		if value == nil {
			note("/identity/"+name, "not observed; identity unknown")
		}
	}
	if r.AuthorizationRef == nil {
		note("/authorization_ref", "no independently reconciled authorization for this run")
	}
	if r.Profile == nil {
		note("/profile", "no run profile declared")
	}
	if r.Limits.SpendUSD == nil {
		note("/limits/spend_usd", "no spend bound declared; unknown is not zero")
	}
}

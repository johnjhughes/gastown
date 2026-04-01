package beads

import (
	"testing"
)

func TestHandoffBeadTitle(t *testing.T) {
	tests := []struct {
		role string
		want string
	}{
		{"mayor", "mayor Handoff"},
		{"deacon", "deacon Handoff"},
		{"gastown/witness", "gastown/witness Handoff"},
		{"gastown/crew/joe", "gastown/crew/joe Handoff"},
		{"", " Handoff"},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := HandoffBeadTitle(tt.role)
			if got != tt.want {
				t.Errorf("HandoffBeadTitle(%q) = %q, want %q", tt.role, got, tt.want)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	// Verify the status constants haven't changed (these are used in protocol)
	if StatusPinned != "pinned" {
		t.Errorf("StatusPinned = %q, want %q", StatusPinned, "pinned")
	}
	if StatusHooked != "hooked" {
		t.Errorf("StatusHooked = %q, want %q", StatusHooked, "hooked")
	}
}

func TestBuildAttachmentFieldsForFormulaReference(t *testing.T) {
	existing := &AttachmentFields{
		AttachedMolecule: "gt-wisp-old",
		DispatchedBy:     "mayor",
		AttachedVars:     []string{"mode=patrol"},
	}

	fields := buildAttachmentFields(existing, "mol-witness-patrol")
	if fields.AttachedMolecule != "" {
		t.Fatalf("AttachedMolecule = %q, want empty for formula refs", fields.AttachedMolecule)
	}
	if fields.AttachedFormula != "mol-witness-patrol" {
		t.Fatalf("AttachedFormula = %q, want mol-witness-patrol", fields.AttachedFormula)
	}
	if fields.DispatchedBy != "mayor" {
		t.Fatalf("DispatchedBy = %q, want mayor", fields.DispatchedBy)
	}
	if len(fields.AttachedVars) != 1 || fields.AttachedVars[0] != "mode=patrol" {
		t.Fatalf("AttachedVars = %#v, want preserved vars", fields.AttachedVars)
	}
	if fields.AttachedAt == "" {
		t.Fatal("AttachedAt should be populated")
	}
}

func TestBuildAttachmentFieldsForMoleculePreservesFormula(t *testing.T) {
	existing := &AttachmentFields{AttachedFormula: "mol-witness-patrol"}
	fields := buildAttachmentFields(existing, "awt-wisp-123")
	if fields.AttachedMolecule != "awt-wisp-123" {
		t.Fatalf("AttachedMolecule = %q, want awt-wisp-123", fields.AttachedMolecule)
	}
	if fields.AttachedFormula != "mol-witness-patrol" {
		t.Fatalf("AttachedFormula = %q, want preserved formula", fields.AttachedFormula)
	}
}

func TestNormalizeFormulaAttachmentRef(t *testing.T) {
	if got := normalizeFormulaAttachmentRef("mol-witness-patrol.formula.toml"); got != "mol-witness-patrol" {
		t.Fatalf("normalizeFormulaAttachmentRef() = %q, want mol-witness-patrol", got)
	}
	if !isFormulaAttachmentRef("mol-witness-patrol") {
		t.Fatal("isFormulaAttachmentRef should detect mol-witness-patrol")
	}
	if isFormulaAttachmentRef("awt-wisp-123") {
		t.Fatal("isFormulaAttachmentRef should reject runtime wisp IDs")
	}
}

func TestCurrentTimestamp(t *testing.T) {
	ts := currentTimestamp()
	if ts == "" {
		t.Fatal("currentTimestamp() returned empty string")
	}
	// Should be RFC3339 format
	if len(ts) < 20 {
		t.Errorf("timestamp too short: %q (expected RFC3339)", ts)
	}
	// Should contain T separator and Z suffix (UTC)
	found := false
	for _, c := range ts {
		if c == 'T' {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("timestamp missing T separator: %q", ts)
	}
}

func TestClearMailResultZeroValues(t *testing.T) {
	// Verify zero-value struct is safe to use
	result := &ClearMailResult{}
	if result.Closed != 0 || result.Cleared != 0 {
		t.Errorf("expected zero values, got Closed=%d Cleared=%d", result.Closed, result.Cleared)
	}
}

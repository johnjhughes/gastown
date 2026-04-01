package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/formula"
)

// HookAttachmentValidCheck verifies that attached molecules exist and are not closed.
// This detects when a hook's attached_molecule field points to a non-existent or
// closed issue, which can leave agents with stale work assignments.
type HookAttachmentValidCheck struct {
	FixableCheck
	invalidAttachments       []invalidAttachment
	legacyFormulaAttachments []legacyFormulaAttachment
}

type invalidAttachment struct {
	pinnedBeadID  string
	pinnedBeadDir string // Directory where the pinned bead was found
	moleculeID    string
	reason        string // "not_found" or "closed"
}

type legacyFormulaAttachment struct {
	pinnedBeadID  string
	pinnedBeadDir string // Directory where the pinned bead was found
	formulaName   string
}

type hookAttachmentScanResult struct {
	invalid []invalidAttachment
	legacy  []legacyFormulaAttachment
}

// NewHookAttachmentValidCheck creates a new hook attachment validation check.
func NewHookAttachmentValidCheck() *HookAttachmentValidCheck {
	return &HookAttachmentValidCheck{
		FixableCheck: FixableCheck{
			BaseCheck: BaseCheck{
				CheckName:        "hook-attachment-valid",
				CheckDescription: "Verify attached molecules exist and are not closed",
				CheckCategory:    CategoryHooks,
			},
		},
	}
}

// Run checks all pinned beads for invalid molecule attachments.
func (c *HookAttachmentValidCheck) Run(ctx *CheckContext) *CheckResult {
	c.invalidAttachments = nil
	c.legacyFormulaAttachments = nil

	var details []string

	// Check town-level beads
	townBeadsDir := filepath.Join(ctx.TownRoot, ".beads")
	townScan := c.checkBeadsDir(townBeadsDir, ctx.TownRoot)
	for _, inv := range townScan.invalid {
		details = append(details, c.formatInvalid(inv))
	}
	for _, legacy := range townScan.legacy {
		details = append(details, c.formatLegacyFormula(legacy))
	}
	c.invalidAttachments = append(c.invalidAttachments, townScan.invalid...)
	c.legacyFormulaAttachments = append(c.legacyFormulaAttachments, townScan.legacy...)

	// Check rig-level beads
	rigDirs := c.findRigBeadsDirs(ctx.TownRoot)
	for _, rigDir := range rigDirs {
		rigScan := c.checkBeadsDir(rigDir, ctx.TownRoot)
		for _, inv := range rigScan.invalid {
			details = append(details, c.formatInvalid(inv))
		}
		for _, legacy := range rigScan.legacy {
			details = append(details, c.formatLegacyFormula(legacy))
		}
		c.invalidAttachments = append(c.invalidAttachments, rigScan.invalid...)
		c.legacyFormulaAttachments = append(c.legacyFormulaAttachments, rigScan.legacy...)
	}

	if len(c.invalidAttachments) > 0 {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusError,
			Message: fmt.Sprintf("Found %d invalid hook attachment(s)", len(c.invalidAttachments)),
			Details: details,
			FixHint: "Run 'gt doctor --fix' to detach invalid molecules, or 'gt mol detach <pinned-bead-id>' manually",
		}
	}

	if len(c.legacyFormulaAttachments) > 0 {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusWarning,
			Message: fmt.Sprintf("Found %d legacy formula attachment(s)", len(c.legacyFormulaAttachments)),
			Details: details,
			FixHint: "Run 'gt doctor --fix' to migrate formula references from attached_molecule to attached_formula",
		}
	}

	return &CheckResult{
		Name:    c.Name(),
		Status:  StatusOK,
		Message: "All hook attachments are valid",
	}
}

// checkBeadsDir checks all pinned beads in a directory for invalid attachments.
func (c *HookAttachmentValidCheck) checkBeadsDir(beadsDir, townRoot string) hookAttachmentScanResult {
	var result hookAttachmentScanResult

	b := beads.New(filepath.Dir(beadsDir))

	// List all pinned beads
	pinnedBeads, err := b.List(beads.ListOptions{
		Status:   beads.StatusPinned,
		Priority: -1,
	})
	if err != nil {
		// Can't list pinned beads - silently skip this directory
		return result
	}

	for _, pinnedBead := range pinnedBeads {
		// Parse attachment fields from the pinned bead
		attachment := beads.ParseAttachmentFields(pinnedBead)
		if attachment == nil || attachment.AttachedMolecule == "" {
			continue // No attachment, skip
		}

		// Verify the attached molecule exists and is not closed
		molecule, err := b.Show(attachment.AttachedMolecule)
		if err != nil {
			// Legacy handoff beads sometimes persisted formula names in
			// attached_molecule. Those are workflow references, not broken
			// molecule pointers, and doctor should migrate rather than flag them
			// as invalid missing issues.
			if c.isFormulaReference(attachment.AttachedMolecule, beadsDir, townRoot) {
				result.legacy = append(result.legacy, legacyFormulaAttachment{
					pinnedBeadID:  pinnedBead.ID,
					pinnedBeadDir: beadsDir,
					formulaName:   normalizeFormulaName(attachment.AttachedMolecule),
				})
				continue
			}

			result.invalid = append(result.invalid, invalidAttachment{
				pinnedBeadID:  pinnedBead.ID,
				pinnedBeadDir: beadsDir,
				moleculeID:    attachment.AttachedMolecule,
				reason:        "not_found",
			})
			continue
		}

		if molecule.Status == "closed" {
			result.invalid = append(result.invalid, invalidAttachment{
				pinnedBeadID:  pinnedBead.ID,
				pinnedBeadDir: beadsDir,
				moleculeID:    attachment.AttachedMolecule,
				reason:        "closed",
			})
		}
	}

	return result
}

// findRigBeadsDirs finds all rig-level .beads directories.
func (c *HookAttachmentValidCheck) findRigBeadsDirs(townRoot string) []string {
	var dirs []string

	// Look for .beads directories in rig subdirectories
	// Pattern: <townRoot>/<rig>/.beads (but NOT <townRoot>/.beads which is town-level)
	cmd := exec.Command("find", townRoot, "-maxdepth", "2", "-type", "d", "-name", ".beads")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		// Skip town-level .beads
		if line == filepath.Join(townRoot, ".beads") {
			continue
		}
		// Skip mayor directory
		if strings.Contains(line, "/mayor/") {
			continue
		}
		dirs = append(dirs, line)
	}

	return dirs
}

// formatInvalid formats an invalid attachment for display.
func (c *HookAttachmentValidCheck) formatInvalid(inv invalidAttachment) string {
	reasonText := "not found"
	if inv.reason == "closed" {
		reasonText = "is closed"
	}
	return fmt.Sprintf("%s: attached molecule %s %s", inv.pinnedBeadID, inv.moleculeID, reasonText)
}

func (c *HookAttachmentValidCheck) formatLegacyFormula(legacy legacyFormulaAttachment) string {
	return fmt.Sprintf("%s: formula %s stored in attached_molecule (legacy format)",
		legacy.pinnedBeadID, legacy.formulaName)
}

func (c *HookAttachmentValidCheck) isFormulaReference(name, beadsDir, townRoot string) bool {
	normalized := normalizeFormulaName(name)
	if normalized == "" {
		return false
	}

	if _, err := formula.GetEmbeddedFormulaContent(normalized); err == nil {
		return true
	}

	homeDir, _ := os.UserHomeDir()
	searchDirs := []string{
		filepath.Join(filepath.Dir(beadsDir), ".beads", "formulas"),
		filepath.Join(townRoot, ".beads", "formulas"),
	}
	if homeDir != "" {
		searchDirs = append(searchDirs, filepath.Join(homeDir, ".beads", "formulas"))
	}

	filename := normalized + ".formula.toml"
	for _, dir := range searchDirs {
		if dir == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, filename)); err == nil {
			return true
		}
	}

	return false
}

func normalizeFormulaName(name string) string {
	return strings.TrimSuffix(name, ".formula.toml")
}

func migrateLegacyFormulaAttachmentFields(issue *beads.Issue, formulaName string) string {
	fields := beads.ParseAttachmentFields(issue)
	if fields == nil {
		fields = &beads.AttachmentFields{}
	}

	normalized := normalizeFormulaName(formulaName)
	if fields.AttachedFormula == "" {
		fields.AttachedFormula = normalized
	}
	if normalizeFormulaName(fields.AttachedMolecule) == normalized {
		fields.AttachedMolecule = ""
	}

	return beads.SetAttachmentFields(issue, fields)
}

// Fix detaches all invalid molecule attachments.
func (c *HookAttachmentValidCheck) Fix(ctx *CheckContext) error {
	var errors []string

	for _, inv := range c.invalidAttachments {
		b := beads.New(filepath.Dir(inv.pinnedBeadDir))

		_, err := b.DetachMolecule(inv.pinnedBeadID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to detach from %s: %v", inv.pinnedBeadID, err))
		}
	}

	for _, legacy := range c.legacyFormulaAttachments {
		b := beads.New(filepath.Dir(legacy.pinnedBeadDir))

		issue, err := b.Show(legacy.pinnedBeadID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to load %s for formula migration: %v", legacy.pinnedBeadID, err))
			continue
		}

		newDesc := migrateLegacyFormulaAttachmentFields(issue, legacy.formulaName)
		if err := b.Update(legacy.pinnedBeadID, beads.UpdateOptions{Description: &newDesc}); err != nil {
			errors = append(errors, fmt.Sprintf("failed to migrate formula attachment on %s: %v", legacy.pinnedBeadID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return nil
}

// HookSingletonCheck ensures each agent has at most one handoff bead.
// Detects when multiple pinned beads exist with the same "{role} Handoff" title,
// which can cause confusion about which handoff is authoritative.
type HookSingletonCheck struct {
	FixableCheck
	duplicates []duplicateHandoff
}

type duplicateHandoff struct {
	title    string
	beadsDir string
	beadIDs  []string // All IDs with this title (first one is kept, rest are duplicates)
}

// NewHookSingletonCheck creates a new hook singleton check.
func NewHookSingletonCheck() *HookSingletonCheck {
	return &HookSingletonCheck{
		FixableCheck: FixableCheck{
			BaseCheck: BaseCheck{
				CheckName:        "hook-singleton",
				CheckDescription: "Ensure each agent has at most one handoff bead",
				CheckCategory:    CategoryHooks,
			},
		},
	}
}

// Run checks all pinned beads for duplicate handoff titles.
func (c *HookSingletonCheck) Run(ctx *CheckContext) *CheckResult {
	c.duplicates = nil

	var details []string

	// Check town-level beads
	townBeadsDir := filepath.Join(ctx.TownRoot, ".beads")
	townDups := c.checkBeadsDir(townBeadsDir)
	for _, dup := range townDups {
		details = append(details, c.formatDuplicate(dup))
	}
	c.duplicates = append(c.duplicates, townDups...)

	// Check rig-level beads using the shared helper
	attachCheck := &HookAttachmentValidCheck{}
	rigDirs := attachCheck.findRigBeadsDirs(ctx.TownRoot)
	for _, rigDir := range rigDirs {
		rigDups := c.checkBeadsDir(rigDir)
		for _, dup := range rigDups {
			details = append(details, c.formatDuplicate(dup))
		}
		c.duplicates = append(c.duplicates, rigDups...)
	}

	if len(c.duplicates) == 0 {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: "All handoff beads are unique",
		}
	}

	totalDups := 0
	for _, dup := range c.duplicates {
		totalDups += len(dup.beadIDs) - 1 // Count extras beyond the first
	}

	return &CheckResult{
		Name:    c.Name(),
		Status:  StatusError,
		Message: fmt.Sprintf("Found %d duplicate handoff bead(s)", totalDups),
		Details: details,
		FixHint: "Run 'gt doctor --fix' to close duplicates, or 'bd close <id>' manually",
	}
}

// checkBeadsDir checks for duplicate handoff beads in a directory.
func (c *HookSingletonCheck) checkBeadsDir(beadsDir string) []duplicateHandoff {
	var duplicates []duplicateHandoff

	b := beads.New(filepath.Dir(beadsDir))

	// List all pinned beads
	pinnedBeads, err := b.List(beads.ListOptions{
		Status:   beads.StatusPinned,
		Priority: -1,
	})
	if err != nil {
		return nil
	}

	// Group pinned beads by title (only those matching "{role} Handoff" pattern)
	titleToIDs := make(map[string][]string)
	for _, bead := range pinnedBeads {
		// Check if title matches handoff pattern (ends with " Handoff")
		if strings.HasSuffix(bead.Title, " Handoff") {
			titleToIDs[bead.Title] = append(titleToIDs[bead.Title], bead.ID)
		}
	}

	// Find duplicates (titles with more than one bead)
	for title, ids := range titleToIDs {
		if len(ids) > 1 {
			duplicates = append(duplicates, duplicateHandoff{
				title:    title,
				beadsDir: beadsDir,
				beadIDs:  ids,
			})
		}
	}

	return duplicates
}

// formatDuplicate formats a duplicate handoff for display.
func (c *HookSingletonCheck) formatDuplicate(dup duplicateHandoff) string {
	return fmt.Sprintf("%q has %d beads: %s", dup.title, len(dup.beadIDs), strings.Join(dup.beadIDs, ", "))
}

// Fix closes duplicate handoff beads, keeping the first one.
func (c *HookSingletonCheck) Fix(ctx *CheckContext) error {
	var errors []string

	for _, dup := range c.duplicates {
		b := beads.New(filepath.Dir(dup.beadsDir))

		// Close all but the first bead (keep the oldest/first one)
		toClose := dup.beadIDs[1:]
		if len(toClose) > 0 {
			err := b.CloseWithReason("duplicate handoff bead", toClose...)
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to close duplicates for %q: %v", dup.title, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return nil
}

// OrphanedAttachmentsCheck detects handoff beads for agents that no longer exist.
// This happens when a polecat worktree is deleted but its handoff bead remains,
// leaving molecules attached to non-existent agents.
type OrphanedAttachmentsCheck struct {
	BaseCheck
	orphans []orphanedHandoff
}

type orphanedHandoff struct {
	beadID    string
	beadTitle string
	beadsDir  string
	agent     string // Parsed agent identity
}

// NewOrphanedAttachmentsCheck creates a new orphaned attachments check.
func NewOrphanedAttachmentsCheck() *OrphanedAttachmentsCheck {
	return &OrphanedAttachmentsCheck{
		BaseCheck: BaseCheck{
			CheckName:        "orphaned-attachments",
			CheckDescription: "Detect handoff beads for non-existent agents",
			CheckCategory:    CategoryHooks,
		},
	}
}

// Run checks all handoff beads for orphaned agents.
func (c *OrphanedAttachmentsCheck) Run(ctx *CheckContext) *CheckResult {
	c.orphans = nil

	var details []string

	// Check town-level beads
	townBeadsDir := filepath.Join(ctx.TownRoot, ".beads")
	townOrphans := c.checkBeadsDir(townBeadsDir, ctx.TownRoot)
	for _, orph := range townOrphans {
		details = append(details, c.formatOrphan(orph))
	}
	c.orphans = append(c.orphans, townOrphans...)

	// Check rig-level beads using the shared helper
	attachCheck := &HookAttachmentValidCheck{}
	rigDirs := attachCheck.findRigBeadsDirs(ctx.TownRoot)
	for _, rigDir := range rigDirs {
		rigOrphans := c.checkBeadsDir(rigDir, ctx.TownRoot)
		for _, orph := range rigOrphans {
			details = append(details, c.formatOrphan(orph))
		}
		c.orphans = append(c.orphans, rigOrphans...)
	}

	if len(c.orphans) == 0 {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: "No orphaned handoff beads found",
		}
	}

	return &CheckResult{
		Name:    c.Name(),
		Status:  StatusWarning,
		Message: fmt.Sprintf("Found %d orphaned handoff bead(s)", len(c.orphans)),
		Details: details,
		FixHint: "Reassign with 'gt sling <id> <agent>', or close with 'bd close <id>'",
	}
}

// checkBeadsDir checks for orphaned handoff beads in a directory.
func (c *OrphanedAttachmentsCheck) checkBeadsDir(beadsDir, townRoot string) []orphanedHandoff {
	var orphans []orphanedHandoff

	b := beads.New(filepath.Dir(beadsDir))

	// List all pinned beads
	pinnedBeads, err := b.List(beads.ListOptions{
		Status:   beads.StatusPinned,
		Priority: -1,
	})
	if err != nil {
		return nil
	}

	for _, bead := range pinnedBeads {
		// Check if title matches handoff pattern (ends with " Handoff")
		if !strings.HasSuffix(bead.Title, " Handoff") {
			continue
		}

		// Extract agent identity from title
		agent := strings.TrimSuffix(bead.Title, " Handoff")
		if agent == "" {
			continue
		}

		// Check if agent worktree exists
		if !c.agentExists(agent, townRoot) {
			orphans = append(orphans, orphanedHandoff{
				beadID:    bead.ID,
				beadTitle: bead.Title,
				beadsDir:  beadsDir,
				agent:     agent,
			})
		}
	}

	return orphans
}

// agentExists checks if an agent's worktree exists.
// Agent identities follow patterns like:
//   - "gastown/nux" → polecat at <townRoot>/gastown/polecats/nux
//   - "gastown/crew/joe" → crew at <townRoot>/gastown/crew/joe
//   - "mayor" → mayor at <townRoot>/mayor
//   - "gastown-witness" → witness at <townRoot>/gastown/witness
//   - "gastown-refinery" → refinery at <townRoot>/gastown/refinery
func (c *OrphanedAttachmentsCheck) agentExists(agent, townRoot string) bool {
	// Handle special roles with hyphen separator
	if strings.HasSuffix(agent, "-witness") {
		rig := strings.TrimSuffix(agent, "-witness")
		path := filepath.Join(townRoot, rig, "witness")
		return dirExists(path)
	}
	if strings.HasSuffix(agent, "-refinery") {
		rig := strings.TrimSuffix(agent, "-refinery")
		path := filepath.Join(townRoot, rig, "refinery")
		return dirExists(path)
	}

	// Handle mayor
	if agent == "mayor" {
		return dirExists(filepath.Join(townRoot, "mayor"))
	}

	// Handle crew (rig/crew/name pattern)
	if strings.Contains(agent, "/crew/") {
		parts := strings.SplitN(agent, "/crew/", 2)
		if len(parts) == 2 {
			path := filepath.Join(townRoot, parts[0], "crew", parts[1])
			return dirExists(path)
		}
	}

	// Handle polecats (rig/name pattern) - most common case
	if strings.Contains(agent, "/") {
		parts := strings.SplitN(agent, "/", 2)
		if len(parts) == 2 {
			path := filepath.Join(townRoot, parts[0], "polecats", parts[1])
			return dirExists(path)
		}
	}

	// Unknown pattern - assume exists to avoid false positives
	return true
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// formatOrphan formats an orphaned handoff for display.
func (c *OrphanedAttachmentsCheck) formatOrphan(orph orphanedHandoff) string {
	return fmt.Sprintf("%s: agent %q no longer exists", orph.beadID, orph.agent)
}

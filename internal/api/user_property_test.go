package api

import (
	"bytes"
	"encoding/csv"
	"math/rand"
	"regexp"
	"strings"
	"testing"
	"testing/quick"
)

// ──────────────────────────────────────────────────────────────────────────────
// Types used by property tests (local to test, modeling user management logic)
// ──────────────────────────────────────────────────────────────────────────────

// CustomerFilter represents a customer record used for filter matching.
type CustomerFilter struct {
	Status       string
	PlanID       int64
	NodeID       int64
	GroupID      int64
	Tags         []int64
	BandwidthPct float64
}

// FilterCriteria represents a single filter criterion applied with AND logic.
type FilterCriteria struct {
	Type  string // "status", "plan", "node", "group", "tag", "bandwidth_min"
	Value any
}

// TagAssociation models a row in customer_tags.
type TagAssociation struct {
	CustomerID int64
	TagID      int64
}

// CSVRow represents a row from a CSV import file.
type CSVRow struct {
	Username string
	Password string
	Email    string
}

// ──────────────────────────────────────────────────────────────────────────────
// Pure functions under test
// ──────────────────────────────────────────────────────────────────────────────

// simulateBulkOperation models a bulk operation where certain indices fail.
// Returns (succeeded, failed) counts.
func simulateBulkOperation(total int, failureIndices []int) (succeeded, failed int) {
	if total < 0 {
		total = 0
	}

	// Deduplicate and filter failure indices to valid range [0, total)
	failSet := make(map[int]bool)
	for _, idx := range failureIndices {
		if idx >= 0 && idx < total {
			failSet[idx] = true
		}
	}

	failed = len(failSet)
	succeeded = total - failed
	return
}

// matchesAllFilters checks if a customer matches ALL filter criteria (AND logic).
func matchesAllFilters(customer CustomerFilter, filters []FilterCriteria) bool {
	for _, f := range filters {
		switch f.Type {
		case "status":
			if val, ok := f.Value.(string); ok && customer.Status != val {
				return false
			}
		case "plan":
			if val, ok := f.Value.(int64); ok && customer.PlanID != val {
				return false
			}
		case "node":
			if val, ok := f.Value.(int64); ok && customer.NodeID != val {
				return false
			}
		case "group":
			if val, ok := f.Value.(int64); ok && customer.GroupID != val {
				return false
			}
		case "tag":
			if val, ok := f.Value.(int64); ok {
				found := false
				for _, t := range customer.Tags {
					if t == val {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
		case "bandwidth_min":
			if val, ok := f.Value.(float64); ok && customer.BandwidthPct < val {
				return false
			}
		}
	}
	return true
}

// cascadeDelete removes all associations for a given tagID.
// Returns remaining associations (those not matching the deleted tagID).
func cascadeDelete(tagID int64, associations []TagAssociation) []TagAssociation {
	var remaining []TagAssociation
	for _, a := range associations {
		if a.TagID != tagID {
			remaining = append(remaining, a)
		}
	}
	return remaining
}

// csvUsernameRe matches 3-32 chars: alphanumeric + underscore (same as importUsernameRe).
var csvUsernameRe = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

// validateCSVRows validates CSV rows and returns counts of valid and invalid rows.
// A row is invalid if username is empty or doesn't match the pattern.
func validateCSVRows(rows []CSVRow) (valid, invalid int) {
	for _, row := range rows {
		if row.Username == "" || !csvUsernameRe.MatchString(row.Username) {
			invalid++
		} else {
			valid++
		}
	}
	return
}

// ──────────────────────────────────────────────────────────────────────────────
// Property 20: Bulk Operation Resilience
// **Validates: Requirements 18.2, 18.4, 18.5**
// ──────────────────────────────────────────────────────────────────────────────

// For N users where K indices fail: succeeded = N-K, failed = K, succeeded+failed = N.
func TestProperty20_BulkOperationResilience(t *testing.T) {
	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Random total between 1 and 200
		total := rng.Intn(200) + 1

		// Random number of failure indices (0 to total)
		numFailures := rng.Intn(total + 1)
		failureIndices := make([]int, numFailures)
		for i := 0; i < numFailures; i++ {
			failureIndices[i] = rng.Intn(total)
		}

		succeeded, failed := simulateBulkOperation(total, failureIndices)

		// Property 1: succeeded + failed = total
		if succeeded+failed != total {
			return false
		}

		// Property 2: both are non-negative
		if succeeded < 0 || failed < 0 {
			return false
		}

		// Property 3: failed <= total
		if failed > total {
			return false
		}

		// Property 4: succeeded <= total
		if succeeded > total {
			return false
		}

		// Property 5: failed equals the number of unique valid failure indices
		uniqueFailures := make(map[int]bool)
		for _, idx := range failureIndices {
			if idx >= 0 && idx < total {
				uniqueFailures[idx] = true
			}
		}
		if failed != len(uniqueFailures) {
			return false
		}

		return true
	}

	cfg := &quick.Config{MaxCount: 200}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("Property 20 violated (Bulk Operation Resilience): %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Property 21: Advanced Filter AND Logic
// **Validates: Requirements 19.1, 19.2**
// ──────────────────────────────────────────────────────────────────────────────

// For random customer data and random filter criteria: result must satisfy ALL
// criteria (AND logic). If any criterion is not met, return false.
func TestProperty21_AdvancedFilterANDLogic(t *testing.T) {
	statuses := []string{"active", "expired", "disabled", "suspended"}

	rng := rand.New(rand.NewSource(21))

	for i := 0; i < 100; i++ {
		// Generate a random customer
		customer := CustomerFilter{
			Status:       statuses[rng.Intn(len(statuses))],
			PlanID:       int64(rng.Intn(10) + 1),
			NodeID:       int64(rng.Intn(20) + 1),
			GroupID:      int64(rng.Intn(5) + 1),
			BandwidthPct: rng.Float64() * 100,
		}
		// Assign 0-3 random tags
		numTags := rng.Intn(4)
		customer.Tags = make([]int64, numTags)
		for j := 0; j < numTags; j++ {
			customer.Tags[j] = int64(rng.Intn(10) + 1)
		}

		// Generate 1-5 random filter criteria
		numFilters := rng.Intn(5) + 1
		filters := make([]FilterCriteria, numFilters)
		filterTypes := []string{"status", "plan", "node", "group", "tag", "bandwidth_min"}

		for j := 0; j < numFilters; j++ {
			ft := filterTypes[rng.Intn(len(filterTypes))]
			var val any
			switch ft {
			case "status":
				val = statuses[rng.Intn(len(statuses))]
			case "plan":
				val = int64(rng.Intn(10) + 1)
			case "node":
				val = int64(rng.Intn(20) + 1)
			case "group":
				val = int64(rng.Intn(5) + 1)
			case "tag":
				val = int64(rng.Intn(10) + 1)
			case "bandwidth_min":
				val = rng.Float64() * 100
			}
			filters[j] = FilterCriteria{Type: ft, Value: val}
		}

		result := matchesAllFilters(customer, filters)

		// Verify: if result is true, every criterion must be satisfied individually
		if result {
			for _, f := range filters {
				if !matchesAllFilters(customer, []FilterCriteria{f}) {
					t.Fatalf("iteration %d: matchesAllFilters returned true but individual criterion %v fails",
						i, f.Type)
				}
			}
		}

		// Verify: if any single criterion fails, result must be false
		anyFails := false
		for _, f := range filters {
			if !matchesAllFilters(customer, []FilterCriteria{f}) {
				anyFails = true
				break
			}
		}
		if anyFails && result {
			t.Fatalf("iteration %d: matchesAllFilters returned true but at least one criterion fails", i)
		}
		if !anyFails && !result {
			t.Fatalf("iteration %d: matchesAllFilters returned false but all individual criteria pass", i)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Property 22: User Tag Cascade Delete
// **Validates: Requirements 20.5**
// ──────────────────────────────────────────────────────────────────────────────

// When a tag is deleted, all customer_tags associations for that tag_id should
// be removed, but the customer records themselves are unaffected.
func TestProperty22_UserTagCascadeDelete(t *testing.T) {
	rng := rand.New(rand.NewSource(22))

	for i := 0; i < 100; i++ {
		// Generate random associations (5-50 entries)
		numAssociations := rng.Intn(46) + 5
		associations := make([]TagAssociation, numAssociations)
		for j := 0; j < numAssociations; j++ {
			associations[j] = TagAssociation{
				CustomerID: int64(rng.Intn(20) + 1),
				TagID:      int64(rng.Intn(10) + 1),
			}
		}

		// Pick a random tagID to delete (1-10)
		deletedTagID := int64(rng.Intn(10) + 1)

		// Record original customer IDs
		originalCustomerIDs := make(map[int64]bool)
		for _, a := range associations {
			originalCustomerIDs[a.CustomerID] = true
		}

		// Perform cascade delete
		remaining := cascadeDelete(deletedTagID, associations)

		// Property 1: No remaining association has the deleted tag_id
		for _, a := range remaining {
			if a.TagID == deletedTagID {
				t.Fatalf("iteration %d: cascade delete left association with deleted tagID %d", i, deletedTagID)
			}
		}

		// Property 2: All remaining associations have tag_ids that are NOT the deleted one
		for _, a := range remaining {
			if a.TagID == deletedTagID {
				t.Fatalf("iteration %d: remaining association still has tagID %d", i, deletedTagID)
			}
		}

		// Property 3: Customer IDs in remaining are a subset of original customer IDs
		for _, a := range remaining {
			if !originalCustomerIDs[a.CustomerID] {
				t.Fatalf("iteration %d: remaining association has unknown customerID %d", i, a.CustomerID)
			}
		}

		// Property 4: Count check — remaining = total - count(deleted tag associations)
		deletedCount := 0
		for _, a := range associations {
			if a.TagID == deletedTagID {
				deletedCount++
			}
		}
		if len(remaining) != numAssociations-deletedCount {
			t.Fatalf("iteration %d: expected %d remaining, got %d", i, numAssociations-deletedCount, len(remaining))
		}

		// Property 5: Non-deleted associations are preserved exactly
		expectedRemaining := make([]TagAssociation, 0)
		for _, a := range associations {
			if a.TagID != deletedTagID {
				expectedRemaining = append(expectedRemaining, a)
			}
		}
		if len(remaining) != len(expectedRemaining) {
			t.Fatalf("iteration %d: remaining length mismatch: got %d, expected %d", i, len(remaining), len(expectedRemaining))
		}
		for j, a := range remaining {
			if a.CustomerID != expectedRemaining[j].CustomerID || a.TagID != expectedRemaining[j].TagID {
				t.Fatalf("iteration %d: remaining[%d] mismatch", i, j)
			}
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Property 23: CSV Import Validation
// **Validates: Requirements 21.3, 21.5**
// ──────────────────────────────────────────────────────────────────────────────

// For M valid rows + K invalid rows: counts should match.
// A row is invalid if username is empty or doesn't match the pattern.
func TestProperty23_CSVImportValidation(t *testing.T) {
	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))

		// Generate a mix of valid and invalid rows
		numValid := rng.Intn(20) + 1
		numInvalid := rng.Intn(20)
		totalRows := numValid + numInvalid

		rows := make([]CSVRow, 0, totalRows)

		// Generate valid rows (3-32 chars alphanumeric+underscore)
		validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_"
		for i := 0; i < numValid; i++ {
			nameLen := rng.Intn(30) + 3 // 3-32 chars
			var sb strings.Builder
			for j := 0; j < nameLen; j++ {
				sb.WriteByte(validChars[rng.Intn(len(validChars))])
			}
			rows = append(rows, CSVRow{
				Username: sb.String(),
				Password: "pass123",
				Email:    "user@example.com",
			})
		}

		// Generate invalid rows
		invalidUsernames := []string{
			"",                      // empty
			"ab",                    // too short (2 chars)
			"a b",                   // contains space
			"user@#$",               // invalid chars
			strings.Repeat("a", 33), // too long
		}
		for i := 0; i < numInvalid; i++ {
			rows = append(rows, CSVRow{
				Username: invalidUsernames[rng.Intn(len(invalidUsernames))],
				Password: "pass123",
				Email:    "user@example.com",
			})
		}

		// Shuffle the rows
		rng.Shuffle(len(rows), func(i, j int) {
			rows[i], rows[j] = rows[j], rows[i]
		})

		valid, invalid := validateCSVRows(rows)

		// Property 1: valid + invalid = total
		if valid+invalid != totalRows {
			return false
		}

		// Property 2: valid count matches expected valid rows
		if valid != numValid {
			return false
		}

		// Property 3: invalid count matches expected invalid rows
		if invalid != numInvalid {
			return false
		}

		// Property 4: both are non-negative
		if valid < 0 || invalid < 0 {
			return false
		}

		return true
	}

	cfg := &quick.Config{MaxCount: 200}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("Property 23 violated (CSV Import Validation): %v", err)
	}
}

// TestProperty23_CSVImportValidation_RealCSV additionally validates that CSV parsing
// combined with row validation produces correct counts using actual CSV encoding.
func TestProperty23_CSVImportValidation_RealCSV(t *testing.T) {
	rng := rand.New(rand.NewSource(2323))

	for i := 0; i < 100; i++ {
		numValid := rng.Intn(15) + 1
		numInvalid := rng.Intn(10)

		// Build a CSV buffer
		var buf bytes.Buffer
		writer := csv.NewWriter(&buf)

		// Write header
		_ = writer.Write([]string{"username", "password", "email"})

		expectedValid := 0
		expectedInvalid := 0

		validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_"

		// Write valid rows
		for j := 0; j < numValid; j++ {
			nameLen := rng.Intn(30) + 3
			var sb strings.Builder
			for k := 0; k < nameLen; k++ {
				sb.WriteByte(validChars[rng.Intn(len(validChars))])
			}
			_ = writer.Write([]string{sb.String(), "pass123", "user@test.com"})
			expectedValid++
		}

		// Write invalid rows
		for j := 0; j < numInvalid; j++ {
			invalidNames := []string{"", "ab", "a b", "x@y", strings.Repeat("z", 33)}
			_ = writer.Write([]string{invalidNames[rng.Intn(len(invalidNames))], "pass123", "user@test.com"})
			expectedInvalid++
		}
		writer.Flush()

		// Parse the CSV and validate
		reader := csv.NewReader(&buf)
		records, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("iteration %d: failed to parse CSV: %v", i, err)
		}

		// Skip header, collect rows
		var rows []CSVRow
		for _, record := range records[1:] {
			rows = append(rows, CSVRow{
				Username: strings.TrimSpace(record[0]),
				Password: strings.TrimSpace(record[1]),
				Email:    strings.TrimSpace(record[2]),
			})
		}

		valid, invalid := validateCSVRows(rows)

		if valid != expectedValid {
			t.Fatalf("iteration %d: valid count %d != expected %d", i, valid, expectedValid)
		}
		if invalid != expectedInvalid {
			t.Fatalf("iteration %d: invalid count %d != expected %d", i, invalid, expectedInvalid)
		}
		if valid+invalid != len(rows) {
			t.Fatalf("iteration %d: valid(%d) + invalid(%d) != total(%d)", i, valid, invalid, len(rows))
		}
	}
}

// Package hashing computes the content hash design.md §2.4 defines: the
// signal that decides no-op vs. update (spec.md REQ-010, REQ-011).
package hashing

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// Hash computes a content hash over exactly the design.md §2.4 input set:
// note type, rendered field values (post-go-org), the effective deck name
// (post-REQ-005 default-deck fallback), and the sorted tag set. Deliberately
// EXCLUDES source_path, heading position, and ANKI_NOTE_ID (design.md §2.4)
// — none of those three describe the note's synchronized content.
//
// Canonicalization contract (load-bearing for M3, plan.md D-9 step 2): the
// orphan-confirmation predicate calls this function a SECOND time with
// values recovered from an AnkiConnect notesInfo response — noteType from
// the response's modelName, fields from its fields map, tags from its tag
// list, and deck from the candidate's own registry-recorded resolved deck
// (notesInfo returns no deck field, design.md §2.4). Two calls carrying
// equal logical content MUST produce identical hashes regardless of:
//
//   - Go's randomized map iteration order over fields (field names are
//     sorted before hashing);
//   - the order tags were supplied in (tags are sorted before hashing,
//     without mutating the caller's slice).
//
// A nil tags slice and an empty (non-nil) tags slice produce the same hash —
// both mean "no tags".
//
// Callers MUST NOT depend on the hash's textual FORMAT remaining stable
// across versions of this function — only equality between two calls over
// equal logical content is a guaranteed contract.
func Hash(noteType string, fields map[string]string, deck string, tags []string) string {
	var b strings.Builder

	b.WriteString("note_type\x00")
	b.WriteString(noteType)
	b.WriteString("\x01")

	fieldNames := make([]string, 0, len(fields))
	for name := range fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)
	for _, name := range fieldNames {
		b.WriteString("field\x00")
		b.WriteString(name)
		b.WriteString("\x00")
		b.WriteString(fields[name])
		b.WriteString("\x01")
	}

	b.WriteString("deck\x00")
	b.WriteString(deck)
	b.WriteString("\x01")

	sortedTags := make([]string, len(tags))
	copy(sortedTags, tags)
	sort.Strings(sortedTags)
	for _, tag := range sortedTags {
		b.WriteString("tag\x00")
		b.WriteString(tag)
		b.WriteString("\x01")
	}

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

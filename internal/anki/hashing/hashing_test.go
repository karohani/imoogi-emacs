package hashing

import "testing"

func TestHash_IdenticalInputsProduceIdenticalHash(t *testing.T) {
	fields := map[string]string{"Front": "<p>Q</p>", "Back": "<p>A</p>"}
	tags := []string{"geography", "europe"}

	h1 := Hash("Basic", fields, "Inbox", tags)
	h2 := Hash("Basic", fields, "Inbox", tags)

	if h1 != h2 {
		t.Fatalf("Hash is not deterministic: %q != %q", h1, h2)
	}
	if h1 == "" {
		t.Fatal("Hash returned empty string")
	}
}

// TestHash_FieldMapOrderIndependence is one of the two highest-risk behaviors
// in this milestone: M3's orphan-confirmation predicate (plan.md D-9 step 2)
// must be able to recompute this hash from a fresh map literal (from an
// AnkiConnect notesInfo response) and get the same result Go's randomized map
// iteration order would otherwise threaten.
func TestHash_FieldMapOrderIndependence(t *testing.T) {
	fieldsA := map[string]string{"Front": "Q", "Back": "A", "Extra": "Z"}
	fieldsB := map[string]string{"Extra": "Z", "Front": "Q", "Back": "A"}

	hA := Hash("Basic", fieldsA, "Inbox", []string{"tag1"})
	hB := Hash("Basic", fieldsB, "Inbox", []string{"tag1"})

	if hA != hB {
		t.Fatalf("Hash depends on field map construction order: %q != %q", hA, hB)
	}
}

// TestHash_TagSliceOrderIndependence is the second highest-risk behavior:
// tag order must never affect the hash (design.md §2.4: "Sorted so tag order
// never spuriously invalidates a hash").
func TestHash_TagSliceOrderIndependence(t *testing.T) {
	fields := map[string]string{"Front": "Q", "Back": "A"}

	h1 := Hash("Basic", fields, "Inbox", []string{"alpha", "beta", "gamma"})
	h2 := Hash("Basic", fields, "Inbox", []string{"gamma", "alpha", "beta"})

	if h1 != h2 {
		t.Fatalf("Hash depends on tag slice order: %q != %q", h1, h2)
	}
}

func TestHash_TagSliceOrderIndependence_DoesNotMutateCallerSlice(t *testing.T) {
	fields := map[string]string{"Front": "Q"}
	tags := []string{"gamma", "alpha", "beta"}
	original := append([]string(nil), tags...)

	Hash("Basic", fields, "Inbox", tags)

	for i := range tags {
		if tags[i] != original[i] {
			t.Fatalf("Hash mutated the caller's tag slice in place: got %v, want %v", tags, original)
		}
	}
}

func TestHash_DifferentNoteType_ProducesDifferentHash(t *testing.T) {
	fields := map[string]string{"Front": "Q", "Back": "A"}
	hBasic := Hash("Basic", fields, "Inbox", nil)
	hCloze := Hash("Cloze", fields, "Inbox", nil)
	if hBasic == hCloze {
		t.Fatal("Hash did not change when note type changed")
	}
}

func TestHash_DifferentFieldValue_ProducesDifferentHash(t *testing.T) {
	fieldsA := map[string]string{"Front": "Q", "Back": "A"}
	fieldsB := map[string]string{"Front": "Q", "Back": "A-changed"}
	hA := Hash("Basic", fieldsA, "Inbox", nil)
	hB := Hash("Basic", fieldsB, "Inbox", nil)
	if hA == hB {
		t.Fatal("Hash did not change when a field value changed")
	}
}

// TestHash_DeckChange_ProducesDifferentHash grounds design.md §2.4: the
// effective (post-fallback) deck name IS a hash input — a deck rename must
// invalidate the hash now that a deck change triggers a card move (D-10).
func TestHash_DeckChange_ProducesDifferentHash(t *testing.T) {
	fields := map[string]string{"Front": "Q", "Back": "A"}
	hInbox := Hash("Basic", fields, "Inbox", nil)
	hOther := Hash("Basic", fields, "OtherDeck", nil)
	if hInbox == hOther {
		t.Fatal("Hash did not change when the effective deck changed")
	}
}

// TestHash_TagContentChange_ProducesDifferentHash: tags ARE a hash input
// (only their order is excluded).
func TestHash_TagContentChange_ProducesDifferentHash(t *testing.T) {
	fields := map[string]string{"Front": "Q"}
	h1 := Hash("Basic", fields, "Inbox", []string{"alpha"})
	h2 := Hash("Basic", fields, "Inbox", []string{"beta"})
	if h1 == h2 {
		t.Fatal("Hash did not change when tag content changed")
	}
}

func TestHash_EmptyTagsVsNilTags_ProduceSameHash(t *testing.T) {
	fields := map[string]string{"Front": "Q"}
	h1 := Hash("Basic", fields, "Inbox", nil)
	h2 := Hash("Basic", fields, "Inbox", []string{})
	if h1 != h2 {
		t.Fatalf("Hash treats nil and empty tag slices differently: %q != %q", h1, h2)
	}
}

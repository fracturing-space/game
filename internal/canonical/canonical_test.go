package canonical

import (
	"fmt"
	"testing"
)

func TestBoundaryValidationHelpers(t *testing.T) {
	t.Parallel()

	if err := ValidateExact("", "campaign id", fmt.Errorf); err == nil {
		t.Fatal("ValidateExact(blank) error = nil, want failure")
	}
	if err := ValidateExact(" camp-1 ", "campaign id", fmt.Errorf); err == nil {
		t.Fatal("ValidateExact(padded) error = nil, want failure")
	}
	if err := ValidateExact("camp-1", "campaign id", fmt.Errorf); err != nil {
		t.Fatalf("ValidateExact(valid) error = %v", err)
	}

	if err := ValidateOptionalExact(" scene-1 ", "scene id", fmt.Errorf); err == nil {
		t.Fatal("ValidateOptionalExact(padded) error = nil, want failure")
	}
	if err := ValidateOptionalExact("", "scene id", fmt.Errorf); err != nil {
		t.Fatalf("ValidateOptionalExact(empty) error = %v", err)
	}
	if err := ValidateOptionalExact("scene-1", "scene id", fmt.Errorf); err != nil {
		t.Fatalf("ValidateOptionalExact(valid) error = %v", err)
	}

	if err := ValidateRelativePath("", "artifact path", fmt.Errorf); err == nil {
		t.Fatal("ValidateRelativePath(blank) error = nil, want failure")
	}
	if err := ValidateRelativePath(" notes.md ", "artifact path", fmt.Errorf); err == nil {
		t.Fatal("ValidateRelativePath(padded) error = nil, want failure")
	}
	if err := ValidateRelativePath("/notes.md", "artifact path", fmt.Errorf); err == nil {
		t.Fatal("ValidateRelativePath(leading slash) error = nil, want failure")
	}
	if err := ValidateRelativePath("notes.md", "artifact path", fmt.Errorf); err != nil {
		t.Fatalf("ValidateRelativePath(valid) error = %v", err)
	}
}

func TestValidateOwnedType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		kind        string
		typ         string
		systemID    string
		systemOwned bool
		wantErr     bool
	}{
		{name: "core valid", kind: "command", typ: "campaign.create"},
		{name: "core sys namespace", kind: "command", typ: "sys.demo.create", wantErr: true},
		{name: "system missing id", kind: "command", typ: "sys.demo.create", systemOwned: true, wantErr: true},
		{name: "system padded id", kind: "event", typ: "sys.demo.created", systemID: " demo ", systemOwned: true, wantErr: true},
		{name: "system mismatch", kind: "event", typ: "sys.other.created", systemID: "demo", systemOwned: true, wantErr: true},
		{name: "system valid", kind: "event", typ: "sys.demo.created", systemID: "demo", systemOwned: true},
		{name: "padded type", kind: "command", typ: " campaign.create ", wantErr: true},
		{name: "blank type", kind: "event", typ: "", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateOwnedType(test.kind, test.typ, test.systemID, test.systemOwned, fmt.Errorf)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateOwnedType() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

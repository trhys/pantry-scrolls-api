package utility_test

import (
	"testing"

	util "github.com/trhys/Recipe-Repo-2/internal/utility"
)

func TestSanitizeSearchQuery(t *testing.T) {
	t.Run("normalizes query", func(t *testing.T) {
		got, err := util.SanitizeSearchQuery("  Chicken Soup  ")
		if err != nil {
			t.Fatalf("SanitizeSearchQuery returned error: %v", err)
		}
		if got != "chicken soup" {
			t.Fatalf("expected normalized query to be %q, got %q", "chicken soup", got)
		}
	})

	t.Run("rejects control chars", func(t *testing.T) {
		if _, err := util.SanitizeSearchQuery("chicken\nsoup"); err == nil {
			t.Fatal("expected error for control character in query")
		}
	})

	t.Run("allows empty after trimming", func(t *testing.T) {
		got, err := util.SanitizeSearchQuery("   ")
		if err != nil {
			t.Fatalf("SanitizeSearchQuery returned error: %v", err)
		}
		if got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
	})
}

func TestSanitizeMessageStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid unread", input: " unread ", want: "unread"},
		{name: "valid read", input: "READ", want: "read"},
		{name: "valid archived", input: "archived", want: "archived"},
		{name: "empty string", input: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantErr: true},
		{name: "invalid", input: "pending", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := util.SanitizeMessageStatus(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("expected status %q, got %q", tt.want, got)
			}
		})
	}
}

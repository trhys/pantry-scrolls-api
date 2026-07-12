package server

import "testing"

func TestSanitizeSearchQuery(t *testing.T) {
	t.Run("normalizes query", func(t *testing.T) {
		got, err := sanitizeSearchQuery("  Chicken Soup  ")
		if err != nil {
			t.Fatalf("sanitizeSearchQuery returned error: %v", err)
		}
		if got != "chicken soup" {
			t.Fatalf("expected normalized query to be %q, got %q", "chicken soup", got)
		}
	})

	t.Run("rejects control chars", func(t *testing.T) {
		if _, err := sanitizeSearchQuery("chicken\nsoup"); err == nil {
			t.Fatal("expected error for control character in query")
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
		{name: "invalid", input: "pending", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizeMessageStatus(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("expected status %q, got %q", tt.want, got)
			}
		})
	}
}

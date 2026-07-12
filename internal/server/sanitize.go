package server

import (
	"fmt"
	"strings"
	"unicode"
)

const maxSearchQueryLength = 100

var messageStatuses = map[string]struct{}{
	"unread":   {},
	"read":     {},
	"archived": {},
}

func sanitizeSearchQuery(raw string) (string, error) {
	query := strings.ToLower(strings.TrimSpace(raw))
	if len(query) > maxSearchQueryLength {
		return "", fmt.Errorf("search query exceeds max length")
	}

	for _, r := range query {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("search query contains control characters")
		}
	}

	return query, nil
}

func sanitizeMessageStatus(raw string) (string, error) {
	status := strings.ToLower(strings.TrimSpace(raw))
	if _, ok := messageStatuses[status]; !ok {
		return "", fmt.Errorf("unsupported message status")
	}

	return status, nil
}

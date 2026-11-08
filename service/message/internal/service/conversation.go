package service

import (
	"strings"

	"github.com/google/uuid"
)

var conversationNamespace = uuid.MustParse("6f1c2a52-3b0e-4b8e-9a51-7d1f0c6e2b11")

func ConversationID(a, b string) string {
	first, second := OrderedPair(a, b)
	return uuid.NewSHA1(conversationNamespace, []byte(first+":"+second)).String()
}

func OrderedPair(a, b string) (string, string) {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
	if a < b {
		return a, b
	}
	return b, a
}

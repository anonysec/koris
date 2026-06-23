package api

import (
	"testing"
)

func TestImportUsernameRe(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid simple", "john_doe", true},
		{"valid min length", "abc", true},
		{"valid max length", "abcdefghijklmnopqrstuvwxyz012345", true}, // 32 chars
		{"valid numbers only", "12345", true},
		{"valid underscores", "a_b_c", true},
		{"valid mixed case", "JohnDoe123", true},
		{"too short", "ab", false},
		{"too long", "abcdefghijklmnopqrstuvwxyz0123456", false}, // 33 chars
		{"has space", "john doe", false},
		{"has dash", "john-doe", false},
		{"has dot", "john.doe", false},
		{"has at sign", "john@doe", false},
		{"empty string", "", false},
		{"special chars", "user!name", false},
		{"unicode chars", "ユーザー名", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := importUsernameRe.MatchString(tt.input)
			if got != tt.want {
				t.Errorf("importUsernameRe.MatchString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestImportEmailRe(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid basic", "user@example.com", true},
		{"valid subdomain", "user@sub.example.com", true},
		{"valid plus addressing", "user+tag@example.com", true},
		{"valid dots in local", "first.last@example.com", true},
		{"missing at sign", "userexample.com", false},
		{"missing domain", "user@", false},
		{"missing local part", "@example.com", false},
		{"space in email", "user @example.com", false},
		{"double at sign", "user@@example.com", false},
		{"empty string", "", false},
		{"missing tld", "user@example", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := importEmailRe.MatchString(tt.input)
			if got != tt.want {
				t.Errorf("importEmailRe.MatchString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

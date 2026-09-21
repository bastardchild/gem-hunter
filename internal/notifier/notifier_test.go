package notifier

import "testing"

func TestEnvelopeAddress(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Gem Hunter <alerts@gemhunter.app>", "alerts@gemhunter.app"},
		{"alerts@gemhunter.app", "alerts@gemhunter.app"},
		{"  Gem Hunter Alerts <noreply@example.com>  ", "noreply@example.com"},
		{"", "alerts@gemhunter.app"},
	}
	for _, tt := range tests {
		if got := envelopeAddress(tt.in); got != tt.want {
			t.Errorf("envelopeAddress(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

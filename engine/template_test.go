package engine

import (
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`\\192.168.20.2:\faisal\template`, `\\192.168.20.2\faisal\template`},
		{`\\192.168.20.2\faisal\template`, `\\192.168.20.2\faisal\template`},
		{`//192.168.20.2/faisal/template`, `\\192.168.20.2\faisal\template`},
		{`"\\192.168.20.2\faisal\template"`, `\\192.168.20.2\faisal\template`},
		{`Z:\faisal\template`, `Z:\faisal\template`},
		{`C:/templates/kain.json`, `C:\templates\kain.json`},
	}

	for _, tt := range tests {
		got := NormalizePath(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

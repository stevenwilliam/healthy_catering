package storage

import "testing"

// The declared content type and the file extension are both attacker-
// controlled. Only the leading bytes decide (99 §7).
func TestDetectTypeReadsTheBytes(t *testing.T) {
	tests := []struct {
		name string
		head []byte
		want string
		ok   bool
	}{
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00}, "image/jpeg", true},
		{"png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, "image/png", true},
		{"pdf", []byte("%PDF-1.7\n..."), "application/pdf", true},
		{"webp", append([]byte("RIFF\x00\x00\x00\x00WEBP"), 'V', 'P'), "image/webp", true},

		// The ones that matter: a script or an executable renamed to .jpg.
		{"shell script", []byte("#!/bin/sh\nrm -rf /"), "", false},
		{"elf binary", []byte{0x7F, 'E', 'L', 'F', 0x02}, "", false},
		{"windows exe", []byte{'M', 'Z', 0x90, 0x00}, "", false},
		{"html with script", []byte("<html><script>alert(1)</script>"), "", false},
		{"empty", nil, "", false},
		{"riff but not webp", []byte("RIFF\x00\x00\x00\x00AVI "), "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := DetectType(tc.head)
			if ok != tc.ok || got != tc.want {
				t.Errorf("DetectType = %q,%v want %q,%v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

// A jpeg header with a .exe name is still a jpeg, and a PE header with a .jpg
// name is still refused — which is the whole point of sniffing.
func TestExtensionComesFromTheDetectedType(t *testing.T) {
	if ext := extensionFor("image/jpeg"); ext != ".jpg" {
		t.Errorf("jpeg extension = %q", ext)
	}
	if ext := extensionFor("application/pdf"); ext != ".pdf" {
		t.Errorf("pdf extension = %q", ext)
	}
}

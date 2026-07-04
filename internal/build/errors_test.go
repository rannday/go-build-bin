package build

import "testing"

func TestValidateVersion(t *testing.T) {
	valid := []string{"1.2.3", "v1", "1.0.0-beta.1", "1.2.3+meta", "1_2_3"}
	for _, version := range valid {
		if err := ValidateVersion(version); err != nil {
			t.Fatalf("ValidateVersion(%q) = %v", version, err)
		}
	}

	invalid := []string{"", ".1.2.3", "1/2/3", "bad space", "../1"}
	for _, version := range invalid {
		if err := ValidateVersion(version); err == nil {
			t.Fatalf("ValidateVersion(%q) expected error", version)
		}
	}
}
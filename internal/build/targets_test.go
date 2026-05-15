package build

import "testing"

func TestDefaultTargets(t *testing.T) {
	got := DefaultTargets()
	if len(got) != 5 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].String() != "windows/amd64:zip" || got[4].String() != "darwin/arm64:tar.gz" {
		t.Fatalf("targets = %#v", got)
	}
}

func TestDefaultTargetStrings(t *testing.T) {
	got := DefaultTargetStrings()
	want := []string{
		"windows/amd64:zip",
		"linux/amd64:tar.gz",
		"linux/arm64:tar.gz",
		"darwin/amd64:tar.gz",
		"darwin/arm64:tar.gz",
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseTarget(t *testing.T) {
	target, err := ParseTarget("linux/amd64")
	if err != nil {
		t.Fatal(err)
	}
	if target.String() != "linux/amd64:tar.gz" {
		t.Fatalf("target = %#v", target)
	}

	target, err = ParseTarget("windows/amd64:zip")
	if err != nil {
		t.Fatal(err)
	}
	if target.String() != "windows/amd64:zip" {
		t.Fatalf("target = %#v", target)
	}
}

func TestArchiveName(t *testing.T) {
	name := ArchiveName("myapp", "1.2.3", TargetSpec{GOOS: "linux", GOARCH: "amd64", Format: "tar.gz"})
	if name != "myapp-1.2.3-linux-amd64.tar.gz" {
		t.Fatalf("name = %q", name)
	}
}

func TestBinaryName(t *testing.T) {
	if got := BinaryName("myapp", "windows"); got != "myapp.exe" {
		t.Fatalf("got %q", got)
	}
	if got := BinaryName("myapp", "linux"); got != "myapp" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateUniqueArchiveNames(t *testing.T) {
	first, _ := ParseTarget("linux/amd64")
	second, _ := ParseTarget("linux/amd64:tar.gz")
	if err := ValidateUniqueArchiveNames("myapp", "1.2.3", []TargetSpec{first, second}); err == nil {
		t.Fatal("expected duplicate error")
	}

	if err := ValidateUniqueArchiveNames("myapp", "1.2.3", []TargetSpec{
		{GOOS: "linux", GOARCH: "amd64", Format: "tar.gz"},
		{GOOS: "linux", GOARCH: "arm64", Format: "tar.gz"},
	}); err != nil {
		t.Fatal(err)
	}
}

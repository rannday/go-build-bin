package build

import "testing"

func TestBuildLdflags(t *testing.T) {
	got := BuildLdflags("1.2.3", "example.com/app.Version", "-buildid=abc", true)
	want := "-s -w -X example.com/app.Version=1.2.3 -buildid=abc"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

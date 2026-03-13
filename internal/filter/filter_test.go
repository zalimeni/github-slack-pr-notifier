package filter

import "testing"

func TestExcerpt(t *testing.T) {
	got := Excerpt("hello\n\nworld")
	if got != "hello world" {
		t.Fatalf("unexpected excerpt: %q", got)
	}
}

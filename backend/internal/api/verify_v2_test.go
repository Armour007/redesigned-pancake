package api

import "testing"

func TestBucketDeterministic(t *testing.T) {
	a1 := bucket("org1", "agent1", "p1")
	a2 := bucket("org1", "agent1", "p1")
	if a1 != a2 {
		t.Fatalf("bucket not deterministic: %d vs %d", a1, a2)
	}
	if a1 < 0 || a1 >= 100 {
		t.Fatalf("bucket out of range: %d", a1)
	}
}

func TestMatchesAllowed(t *testing.T) {
	allowed := []string{"deploy:*", "re:^read:(.*)$", "exact"}
	cases := []struct {
		in   string
		want bool
	}{
		{"deploy:prod", true},
		{"read:foo", true},
		{"exact", true},
		{"nope", false},
	}
	for _, c := range cases {
		if got := matchesAllowed(c.in, allowed); got != c.want {
			t.Fatalf("matchesAllowed(%q)=%v want %v", c.in, got, c.want)
		}
	}
}

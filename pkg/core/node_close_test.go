package core

import "testing"

func TestNodeCloseRunsCleanupOnce(t *testing.T) {
	var calls int
	parent := &Node{}
	child := &Node{OnClose: func() { calls++ }}
	child.WithParent(parent)

	child.Close()
	child.Close()
	if calls != 1 {
		t.Fatalf("cleanup called %d times, want 1", calls)
	}
}

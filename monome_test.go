package monome

import "testing"

func TestOscConnection(t *testing.T) {
	_, err := newOscConnection("localhost:12002")
	if err != nil {
		t.Fatal(err)
	}
}

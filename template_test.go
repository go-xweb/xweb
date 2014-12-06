package xweb

import "testing"

func TestIsNil(t *testing.T) {
	if !IsNil(nil) {
		t.Error("nil")
	}

	if IsNil(1) {
		t.Error("1")
	}

	if IsNil("tttt") {
		t.Error("tttt")
	}

	type A struct {
	}

	var a A

	if IsNil(a) {
		t.Error("a0")
	}

	if IsNil(&a) {
		t.Error("a")
	}

	if IsNil(new(A)) {
		t.Error("a2")
	}

	var b *A
	if !IsNil(b) {
		t.Error("b")
	}

	var c interface{}
	if !IsNil(c) {
		t.Error("c")
	}
}

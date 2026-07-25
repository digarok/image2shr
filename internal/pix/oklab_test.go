package pix

import "testing"

func TestOklabReferencePoints(t *testing.T) {
	// White is (1, 0, 0) and black (0, 0, 0) in Oklab by construction.
	l, a, b := LinearToOklab(1, 1, 1)
	if l < 0.999 || l > 1.001 {
		t.Errorf("white L = %v, want ~1", l)
	}
	if a < -0.001 || a > 0.001 || b < -0.001 || b > 0.001 {
		t.Errorf("white a,b = %v,%v, want ~0", a, b)
	}
	if l, a, b = LinearToOklab(0, 0, 0); l != 0 || a != 0 || b != 0 {
		t.Errorf("black = %v,%v,%v, want 0,0,0", l, a, b)
	}
}

func TestOklabOpponentAxes(t *testing.T) {
	// a: green–red axis (red positive), b: blue–yellow axis (blue negative).
	if _, a, _ := LinearToOklab(1, 0, 0); a <= 0 {
		t.Errorf("red a = %v, want > 0", a)
	}
	if _, a, _ := LinearToOklab(0, 1, 0); a >= 0 {
		t.Errorf("green a = %v, want < 0", a)
	}
	if _, _, b := LinearToOklab(0, 0, 1); b >= 0 {
		t.Errorf("blue b = %v, want < 0", b)
	}
}

func TestOklabGreysAreNeutralAndMonotonic(t *testing.T) {
	prev := -1.0
	for i := 0; i <= 10; i++ {
		v := float64(i) / 10
		l, a, b := LinearToOklab(v, v, v)
		if a < -1e-6 || a > 1e-6 || b < -1e-6 || b > 1e-6 {
			t.Errorf("grey %v: a,b = %v,%v, want 0", v, a, b)
		}
		if l <= prev {
			t.Errorf("grey %v: L = %v not increasing (prev %v)", v, l, prev)
		}
		prev = l
	}
}

package main

import (
	"testing"
	"testing/quick"
)

func TestGroup(t *testing.T) {
	z := ZmodQ{17, 1}
	times := func(x int64) bool {
		return z.Times(x, x) == mod(x+x, 17)
	}
	if err := quick.Check(times, nil); err != nil {
		t.Error(err)
	}

	exp := func(x int64, y int64) bool {
		return z.Exp(x, y) == mod(mod(x, 17)*mod(y, 17), 17)
	}
	if err := quick.Check(exp, nil); err != nil {
		t.Error(err)

	}

	identity := func(x int64) bool {
		return z.Exp(z.G, x) == mod(x+1, 17)
	}
	if err := quick.Check(identity, nil); err != nil {
		t.Error(err)

	}
}

package spotify

import (
	"testing"
	"time"
)

func TestStateGuardGenerateAndConsume(t *testing.T) {
	g := &stateGuard{}

	value, err := g.generate()
	if err != nil {
		t.Fatalf("generate returned error: %v", err)
	}
	if value == "" {
		t.Fatal("generate returned an empty state")
	}

	if !g.consume(value) {
		t.Fatal("consume rejected the freshly generated state")
	}
}

func TestStateGuardGenerateIsRandom(t *testing.T) {
	g := &stateGuard{}

	a, err := g.generate()
	if err != nil {
		t.Fatalf("generate returned error: %v", err)
	}
	b, err := g.generate()
	if err != nil {
		t.Fatalf("generate returned error: %v", err)
	}

	if a == b {
		t.Fatal("two consecutive generate calls returned the same state")
	}
}

func TestStateGuardConsumeIsOneShot(t *testing.T) {
	g := &stateGuard{}

	value, err := g.generate()
	if err != nil {
		t.Fatalf("generate returned error: %v", err)
	}

	if !g.consume(value) {
		t.Fatal("first consume should have succeeded")
	}
	if g.consume(value) {
		t.Fatal("second consume of the same state should have failed")
	}
}

func TestStateGuardConsumeRejectsWrongValue(t *testing.T) {
	g := &stateGuard{}

	if _, err := g.generate(); err != nil {
		t.Fatalf("generate returned error: %v", err)
	}

	if g.consume("not-the-generated-state") {
		t.Fatal("consume accepted a state value it never generated")
	}
}

func TestStateGuardConsumeRejectsMissingValue(t *testing.T) {
	g := &stateGuard{}

	if g.consume("") {
		t.Fatal("consume accepted an empty candidate")
	}
	if g.consume("anything") {
		t.Fatal("consume accepted a candidate when no state was pending")
	}
}

func TestStateGuardConsumeRejectsExpiredValue(t *testing.T) {
	g := &stateGuard{
		value:   "expired-state",
		expires: time.Now().Add(-time.Second),
	}

	if g.consume("expired-state") {
		t.Fatal("consume accepted an expired state")
	}
}

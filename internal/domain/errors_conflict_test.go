package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestConflictErrorsCarryCodeAndField(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", WrapConflict(CodeNodeNameTaken, "name", "a node named lab is already in the cluster"))

	if !IsConflictError(err) {
		t.Fatal("expected a conflict error")
	}
	if got := ErrorCode(err); got != CodeNodeNameTaken {
		t.Errorf("code = %q, want %q", got, CodeNodeNameTaken)
	}
	if got := ErrorField(err); got != "name" {
		t.Errorf("field = %q, want name", got)
	}
	if got := PublicMessage(err); got != "a node named lab is already in the cluster" {
		t.Errorf("public message = %q", got)
	}
}

func TestPlainErrorsHaveNoCodeOrField(t *testing.T) {
	err := errors.New("boom")

	if IsConflictError(err) || IsNotFoundError(err) {
		t.Error("a plain error must not be classified")
	}
	if ErrorCode(err) != "" || ErrorField(err) != "" {
		t.Error("a plain error has no code or field")
	}
}

func TestNodeNotFoundIsNotFound(t *testing.T) {
	if !IsNotFoundError(WrapNodeNotFound("n1", errors.New("sql: no rows"))) {
		t.Fatal("expected a not found error")
	}
}

func TestValidationErrorNamesItsField(t *testing.T) {
	if got := ErrorField(WrapValidationError("api endpoint", errors.New("bad"))); got != "api endpoint" {
		t.Errorf("field = %q", got)
	}
}

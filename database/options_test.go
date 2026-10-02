package database

import (
	"errors"
	"testing"

	"github.com/lucap9056/auth-middleware/database/v2/schema"
)

func TestDefaultOptions_DefaultSchema(t *testing.T) {
	if got, want := defaultOptions().Schema, schema.DefaultParams(); *got != *want {
		t.Errorf("Schema: got %+v, want %+v", *got, *want)
	}
}

func TestWithUserEmailReference_AppliesParsedSchema(t *testing.T) {
	opt, err := WithUserEmailReference("auth.members(mail):citext")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	o := defaultOptions()
	opt(o)

	want := schema.Params{UsersTable: `"auth"."members"`, UsersEmailColumn: `"mail"`, UsersEmailType: "citext"}
	if *o.Schema != want {
		t.Errorf("Schema: got %+v, want %+v", *o.Schema, want)
	}
}

func TestWithUserEmailReference_Invalid(t *testing.T) {
	opt, err := WithUserEmailReference("users")
	if !errors.Is(err, schema.ErrInvalidUserEmailReference) {
		t.Fatalf("err: got %v, want ErrInvalidUserEmailReference", err)
	}
	if opt != nil {
		t.Error("expected nil option on error")
	}
}

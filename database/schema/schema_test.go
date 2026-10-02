package schema

import (
	"errors"
	"strings"
	"testing"
)

func TestDefaultParams_Quoted(t *testing.T) {
	want := Params{UsersTable: `"users"`, UsersEmailColumn: `"email"`, UsersEmailType: "text"}

	if got := DefaultParams(); *got != want {
		t.Errorf("DefaultParams: got %+v, want %+v", got, want)
	}
}

func TestParseUserEmailReference_Valid(t *testing.T) {
	tests := []struct {
		reference string
		want      Params
	}{
		{"users(email)", Params{`"users"`, `"email"`, "text"}},
		{"users(email):citext", Params{`"users"`, `"email"`, "citext"}},
		{"auth.members(mail):varchar(255)", Params{`"auth"."members"`, `"mail"`, "varchar(255)"}},
		{" Users(Email):TEXT ", Params{`"users"`, `"email"`, "text"}},
		{"user(email)", Params{`"user"`, `"email"`, "text"}},
		{"users(email):public.citext", Params{`"users"`, `"email"`, "public.citext"}},
	}

	for _, tt := range tests {
		t.Run(tt.reference, func(t *testing.T) {
			got, err := ParseUserEmailReference(tt.reference)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if *got != tt.want {
				t.Errorf("got %+v, want %+v", *got, tt.want)
			}
		})
	}
}

func TestParseUserEmailReference_Invalid(t *testing.T) {
	references := []string{
		"",
		"users",
		"users.email",
		"users()",
		"(email)",
		"users(email",
		"users(email):",
		"a.b.c(email)",
		"users(email, name)",
		"users(email);DROP TABLE users",
		`users("email")`,
		"users(email):text;DROP TABLE users",
		"users(email):varchar(abc)",
		"1users(email)",
	}

	for _, reference := range references {
		t.Run(reference, func(t *testing.T) {
			params, err := ParseUserEmailReference(reference)
			if !errors.Is(err, ErrInvalidUserEmailReference) {
				t.Fatalf("err: got %v, want ErrInvalidUserEmailReference", err)
			}
			if params != nil {
				t.Error("expected nil params on error")
			}
		})
	}
}

func TestGenerate_Default(t *testing.T) {
	sql, err := Generate(DefaultParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, want := range []string{
		"user_email text NOT NULL",
		`REFERENCES "users" ("email")`,
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("generated schema missing %q:\n%s", want, sql)
		}
	}
	if strings.Contains(sql, "{{") {
		t.Errorf("generated schema contains unrendered template:\n%s", sql)
	}
}

func TestGenerate_CustomReference(t *testing.T) {
	params, err := ParseUserEmailReference("auth.members(mail):varchar(255)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sql, err := Generate(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, want := range []string{
		"user_email varchar(255) NOT NULL",
		`REFERENCES "auth"."members" ("mail")`,
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("generated schema missing %q:\n%s", want, sql)
		}
	}
}

func TestGenerate_NilParams(t *testing.T) {
	if _, err := Generate(nil); err == nil {
		t.Fatal("expected error for nil params, got nil")
	}
}

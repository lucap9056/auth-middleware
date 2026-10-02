package schema

import (
	_ "embed"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/jackc/pgx/v5"
)

//go:embed schema.sql
var schemaSQL string

var schemaTemplate = template.Must(template.New("schema.sql").Parse(schemaSQL))

var ErrInvalidUserEmailReference = errors.New("invalid user email reference, expected <table>(<column>)[:<type>]")

var userEmailReferencePattern = regexp.MustCompile(
	`^(?:([a-z_][a-z0-9_]*)\.)?([a-z_][a-z0-9_]*)\(([a-z_][a-z0-9_]*)\)(?::([a-z_][a-z0-9_]*(?:\.[a-z_][a-z0-9_]*)?(?:\(\d+\))?))?$`,
)

type Params struct {
	UsersTable       string
	UsersEmailColumn string
	UsersEmailType   string
}

func DefaultParams() *Params {
	return &Params{
		UsersTable:       pgx.Identifier{"users"}.Sanitize(),
		UsersEmailColumn: pgx.Identifier{"email"}.Sanitize(),
		UsersEmailType:   "text",
	}
}

func ParseUserEmailReference(reference string) (*Params, error) {
	normalized := strings.ToLower(strings.TrimSpace(reference))

	match := userEmailReferencePattern.FindStringSubmatch(normalized)
	if match == nil {
		return nil, fmt.Errorf("%w: %q", ErrInvalidUserEmailReference, reference)
	}
	schemaName, tableName, column, columnType := match[1], match[2], match[3], match[4]

	table := pgx.Identifier{tableName}
	if schemaName != "" {
		table = pgx.Identifier{schemaName, tableName}
	}

	params := DefaultParams()
	params.UsersTable = table.Sanitize()
	params.UsersEmailColumn = pgx.Identifier{column}.Sanitize()
	if columnType != "" {
		params.UsersEmailType = columnType
	}
	return params, nil
}

func Generate(params *Params) (string, error) {
	var sql strings.Builder
	if err := schemaTemplate.Execute(&sql, params); err != nil {
		return "", err
	}
	return sql.String(), nil
}

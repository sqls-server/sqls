package dialect

type Dialect interface {
	IsIdentifierStart(r rune) bool
	IsIdentifierPart(r rune) bool
	IsDelimitedIdentifierStart(r rune) bool
}

type GenericSQLDialect struct {
}

func (*GenericSQLDialect) IsIdentifierStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '@'
}

func (*GenericSQLDialect) IsIdentifierPart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '@' || r == '_'
}

func (*GenericSQLDialect) IsDelimitedIdentifierStart(r rune) bool {
	return r == '"'
}

var _ Dialect = &GenericSQLDialect{}

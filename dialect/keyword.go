package dialect

type KeywordKind int

//go:generate stringer -type KeywordKind kind.go
const (
	// Matched keyword
	Matched KeywordKind = iota
	// Data Manipulation Language
	DML
	// Data Definition Language
	DDL
	// Data Control Language
	DCL
	// Unmatched keyword (like table and column identifier)
	Unmatched = 99
)

var match = map[string]KeywordKind{
	"ABS":                              Matched,
	"ADD":                              Matched,
	"ASC":                              Matched,
	"ALL":                              Matched,
	"ALLOCATE":                         Matched,
	"ALTER":                            DDL,
	"AND":                              Matched,
	"ANY":                              Matched,
	"ARE":                              Matched,
	"ARRAY":                            Matched,
	"ARRAY_AGG":                        Matched,
	"ARRAY_MAX_CARDINALITY":            Matched,
	"AS":                               Matched,
	"ASENSITIVE":                       Matched,
	"ASYMMETRIC":                       Matched,
	"AT":                               Matched,
	"ATOMIC":                           Matched,
	"AUTHORIZATION":                    Matched,
	"AVG":                              Matched,
	"BEGIN":                            Matched,
	"BEGIN_FRAME":                      Matched,
	"BEGIN_PARTITION":                  Matched,
	"BETWEEN":                          Matched,
	"BIGINT":                           Matched,
	"BINARY":                           Matched,
	"BLOB":                             Matched,
	"BOOLEAN":                          Matched,
	"BOTH":                             Matched,
	"BY":                               Matched,
	"BYTEA":                            Matched,
	"CALL":                             Matched,
	"CALLED":                           Matched,
	"CARDINALITY":                      Matched,
	"CASCADED":                         Matched,
	"CASE":                             Matched,
	"CAST":                             Matched,
	"CEIL":                             Matched,
	"CEILING":                          Matched,
	"CHR":                              Matched,
	"CHAR":                             Matched,
	"CHAR_LENGTH":                      Matched,
	"CHARACTER":                        Matched,
	"CHARACTER_LENGTH":                 Matched,
	"CHECK":                            Matched,
	"CLOB":                             Matched,
	"CLOSE":                            Matched,
	"COALESCE":                         Matched,
	"COLLATE":                          Matched,
	"COLLECT":                          Matched,
	"COLUMN":                           Matched,
	"COMMIT":                           DML,
	"CONDITION":                        Matched,
	"CONNECT":                          Matched,
	"CONSTRAINT":                       Matched,
	"CONTAINS":                         Matched,
	"CONVERT":                          Matched,
	"COPY":                             Matched,
	"CORR":                             Matched,
	"CORRESPONDING":                    Matched,
	"COUNT":                            Matched,
	"COVAR_POP":                        Matched,
	"COVAR_SAMP":                       Matched,
	"CREATE":                           DDL,
	"CROSS":                            Matched,
	"CSV":                              Matched,
	"CUBE":                             Matched,
	"CUME_DIST":                        Matched,
	"CURRENT":                          Matched,
	"CURRENT_CATALOG":                  Matched,
	"CURRENT_DATE":                     Matched,
	"CURRENT_DEFAULT_TRANSFORM_GROUP":  Matched,
	"CURRENT_PATH":                     Matched,
	"CURRENT_ROLE":                     Matched,
	"CURRENT_ROW":                      Matched,
	"CURRENT_SCHEMA":                   Matched,
	"CURRENT_TIME":                     Matched,
	"CURRENT_TIMESTAMP":                Matched,
	"CURRENT_TRANSFORM_GROUP_FOR_TYPE": Matched,
	"CURRENT_USER":                     Matched,
	"CURSOR":                           Matched,
	"CYCLE":                            Matched,
	"DATE":                             Matched,
	"DAY":                              Matched,
	"DEALLOCATE":                       Matched,
	"DEC":                              Matched,
	"DECIMAL":                          Matched,
	"DECLARE":                          Matched,
	"DEFAULT":                          Matched,
	"DELETE":                           DML,
	"DENSE_RANK":                       Matched,
	"DEREF":                            Matched,
	"DESC":                             Matched,
	"DESCRIBE":                         Matched,
	"DETERMINISTIC":                    Matched,
	"DISCONNECT":                       Matched,
	"DISTINCT":                         Matched,
	"DOUBLE":                           Matched,
	"DROP":                             DDL,
	"DYNAMIC":                          Matched,
	"EACH":                             Matched,
	"ELEMENT":                          Matched,
	"ELSE":                             Matched,
	"END":                              Matched,
	"END_FRAME":                        Matched,
	"END_PARTITION":                    Matched,
	"EQUALS":                           Matched,
	"ESCAPE":                           Matched,
	"EVERY":                            Matched,
	"EXCEPT":                           Matched,
	"EXEC":                             Matched,
	"EXECUTE":                          Matched,
	"EXISTS":                           Matched,
	"EXP":                              Matched,
	"EXTERNAL":                         Matched,
	"EXTRACT":                          Matched,
	"FALSE":                            Matched,
	"FETCH":                            Matched,
	"FILTER":                           Matched,
	"FIRST_VALUE":                      Matched,
	"FLOAT":                            Matched,
	"FLOOR":                            Matched,
	"FOLLOWING":                        Matched,
	"FOR":                              Matched,
	"FOREIGN":                          Matched,
	"FRAME_ROW":                        Matched,
	"FREE":                             Matched,
	"FROM":                             Matched,
	"FULL":                             Matched,
	"FUNCTION":                         Matched,
	"FUSION":                           Matched,
	"GET":                              Matched,
	"GLOBAL":                           Matched,
	"GRANT":                            Matched,
	"GROUP":                            Matched,
	"GROUPING":                         Matched,
	"GROUPS":                           Matched,
	"HAVING":                           Matched,
	"HEADER":                           Matched,
	"HOLD":                             Matched,
	"HOUR":                             Matched,
	"IDENTITY":                         Matched,
	"IN":                               Matched,
	"INDICATOR":                        Matched,
	"INNER":                            Matched,
	"INOUT":                            Matched,
	"INSENSITIVE":                      Matched,
	"INSERT":                           DML,
	"INT":                              Matched,
	"INTEGER":                          Matched,
	"INTERSECT":                        Matched,
	"INTERSECTION":                     Matched,
	"INTERVAL":                         Matched,
	"INTO":                             Matched,
	"IS":                               Matched,
	"JOIN":                             Matched,
	"KEY":                              Matched,
	"LAG":                              Matched,
	"LANGUAGE":                         Matched,
	"LARGE":                            Matched,
	"LAST_VALUE":                       Matched,
	"LATERAL":                          Matched,
	"LEAD":                             Matched,
	"LEADING":                          Matched,
	"LEFT":                             Matched,
	"LIKE":                             Matched,
	"LIKE_REGEX":                       Matched,
	"LIMIT":                            Matched,
	"LN":                               Matched,
	"LOCAL":                            Matched,
	"LOCALTIME":                        Matched,
	"LOCALTIMESTAMP":                   Matched,
	"LOCATION":                         Matched,
	"LOWER":                            Matched,
	"MATCH":                            Matched,
	"MATERIALIZED":                     Matched,
	"MAX":                              Matched,
	"MEMBER":                           Matched,
	"MERGE":                            DML,
	"METHOD":                           Matched,
	"MIN":                              Matched,
	"MINUTE":                           Matched,
	"MOD":                              Matched,
	"MODIFIES":                         Matched,
	"MODULE":                           Matched,
	"MONTH":                            Matched,
	"MULTISET":                         Matched,
	"NATIONAL":                         Matched,
	"NATURAL":                          Matched,
	"NCHAR":                            Matched,
	"NCLOB":                            Matched,
	"NEW":                              Matched,
	"NO":                               Matched,
	"NONE":                             Matched,
	"NORMALIZE":                        Matched,
	"NOT":                              Matched,
	"NTH_VALUE":                        Matched,
	"NTILE":                            Matched,
	"NULL":                             Matched,
	"NULLIF":                           Matched,
	"NUMERIC":                          Matched,
	"OBJECT":                           Matched,
	"OCTET_LENGTH":                     Matched,
	"OCCURRENCES_REGEX":                Matched,
	"OF":                               Matched,
	"OFFSET":                           Matched,
	"OLD":                              Matched,
	"ON":                               Matched,
	"ONLY":                             Matched,
	"OPEN":                             Matched,
	"OR":                               Matched,
	"ORDER":                            Matched,
	"OUT":                              Matched,
	"OUTER":                            Matched,
	"OVER":                             Matched,
	"OVERLAPS":                         Matched,
	"OVERLAY":                          Matched,
	"PARAMETER":                        Matched,
	"PARTITION":                        Matched,
	"PARQUET":                          Matched,
	"PERCENT":                          Matched,
	"PERCENT_RANK":                     Matched,
	"PERCENTILE_CONT":                  Matched,
	"PERCENTILE_DISC":                  Matched,
	"PERIOD":                           Matched,
	"PORTION":                          Matched,
	"POSITION":                         Matched,
	"POSITION_REGEX":                   Matched,
	"POWER":                            Matched,
	"PRECEDES":                         Matched,
	"PRECEDING":                        Matched,
	"PRECISION":                        Matched,
	"PREPARE":                          Matched,
	"PRIMARY":                          Matched,
	"PROCEDURE":                        Matched,
	"RANGE":                            Matched,
	"RANK":                             Matched,
	"READS":                            Matched,
	"REAL":                             Matched,
	"RECURSIVE":                        Matched,
	"REF":                              Matched,
	"REFERENCES":                       Matched,
	"REFERENCING":                      Matched,
	"REGCLASS":                         Matched,
	"REGR_AVGX":                        Matched,
	"REGR_AVGY":                        Matched,
	"REGR_COUNT":                       Matched,
	"REGR_INTERCEPT":                   Matched,
	"REGR_R2":                          Matched,
	"REGR_SLOPE":                       Matched,
	"REGR_SXX":                         Matched,
	"REGR_SXY":                         Matched,
	"REGR_SYY":                         Matched,
	"RELEASE":                          Matched,
	"REPLACE":                          DML,
	"RESULT":                           Matched,
	"RETURN":                           Matched,
	"RETURNS":                          Matched,
	"REVOKE":                           Matched,
	"RIGHT":                            Matched,
	"ROLLBACK":                         DML,
	"ROLLUP":                           Matched,
	"ROW":                              Matched,
	"ROW_NUMBER":                       Matched,
	"ROWS":                             Matched,
	"SAVEPOINT":                        Matched,
	"SCOPE":                            Matched,
	"SCROLL":                           Matched,
	"SEARCH":                           Matched,
	"SECOND":                           Matched,
	"SELECT":                           DML,
	"SENSITIVE":                        Matched,
	"SESSION_USER":                     Matched,
	"SET":                              Matched,
	"SIMILAR":                          Matched,
	"SMALLINT":                         Matched,
	"SOME":                             Matched,
	"SPECIFIC":                         Matched,
	"SPECIFICTYPE":                     Matched,
	"SQL":                              Matched,
	"SQLEXCEPTION":                     Matched,
	"SQLSTATE":                         Matched,
	"SQLWARNING":                       Matched,
	"SQRT":                             Matched,
	"START":                            DML,
	"STATIC":                           Matched,
	"STDDEV_POP":                       Matched,
	"STDDEV_SAMP":                      Matched,
	"STDIN":                            Matched,
	"STORED":                           Matched,
	"SUBMULTISET":                      Matched,
	"SUBSTRING":                        Matched,
	"SUBSTRING_REGEX":                  Matched,
	"SUCCEEDS":                         Matched,
	"SUM":                              Matched,
	"SYMMETRIC":                        Matched,
	"SYSTEM":                           Matched,
	"SYSTEM_TIME":                      Matched,
	"SYSTEM_USER":                      Matched,
	"TABLE":                            Matched,
	"TABLESAMPLE":                      Matched,
	"TEXT":                             Matched,
	"THEN":                             Matched,
	"TIME":                             Matched,
	"TIMESTAMP":                        Matched,
	"TIMEZONE_HOUR":                    Matched,
	"TIMEZONE_MINUTE":                  Matched,
	"TO":                               Matched,
	"TRAILING":                         Matched,
	"TRANSLATE":                        Matched,
	"TRANSLATE_REGEX":                  Matched,
	"TRANSLATION":                      Matched,
	"TREAT":                            Matched,
	"TRIGGER":                          Matched,
	"TRUNCATE":                         Matched,
	"TRIM":                             Matched,
	"TRIM_ARRAY":                       Matched,
	"TRUE":                             Matched,
	"UESCAPE":                          Matched,
	"UNBOUNDED":                        Matched,
	"UNION":                            Matched,
	"UNIQUE":                           Matched,
	"UNKNOWN":                          Matched,
	"UNNEST":                           Matched,
	"UPDATE":                           DML,
	"UPPER":                            Matched,
	"UPSERT":                           DML,
	"USER":                             Matched,
	"USING":                            Matched,
	"UUID":                             Matched,
	"VALUE":                            Matched,
	"VALUES":                           Matched,
	"VALUE_OF":                         Matched,
	"VAR_POP":                          Matched,
	"VAR_SAMP":                         Matched,
	"VARBINARY":                        Matched,
	"VARCHAR":                          Matched,
	"VARYING":                          Matched,
	"VERSIONING":                       Matched,
	"VIEW":                             Matched,
	"WHEN":                             Matched,
	"WHENEVER":                         Matched,
	"WHERE":                            Matched,
	"WIDTH_BUCKET":                     Matched,
	"WINDOW":                           Matched,
	"WITH":                             Matched,
	"WITHIN":                           Matched,
	"WITHOUT":                          Matched,
	"YEAR":                             Matched,
	"ZONE":                             Matched,
}

func MatchKeyword(upperWord string) KeywordKind {
	kind, ok := match[upperWord]
	if !ok {
		return Unmatched
	}
	return kind
}

type DatabaseDriver string

const (
	DatabaseDriverMySQL      DatabaseDriver = "mysql"
	DatabaseDriverMySQL8     DatabaseDriver = "mysql8"
	DatabaseDriverMySQL57    DatabaseDriver = "mysql57"
	DatabaseDriverMySQL56    DatabaseDriver = "mysql56"
	DatabaseDriverPostgreSQL DatabaseDriver = "postgresql"
	DatabaseDriverSQLite3    DatabaseDriver = "sqlite3"
	DatabaseDriverMssql      DatabaseDriver = "mssql"
	DatabaseDriverOracle     DatabaseDriver = "oracle"
	DatabaseDriverH2         DatabaseDriver = "h2"
	DatabaseDriverVertica    DatabaseDriver = "vertica"
	DatabaseDriverClickhouse DatabaseDriver = "clickhouse"
)

func DataBaseKeywords(driver DatabaseDriver) []string {
	switch driver {
	case DatabaseDriverMySQL:
		return mysql8Keyword
	case DatabaseDriverMySQL8:
		return mysql8Keyword
	case DatabaseDriverMySQL57:
		return mysql57Keyword
	case DatabaseDriverMySQL56:
		return mysql56Keyword
	case DatabaseDriverPostgreSQL:
		return postgresql13Keywords
	case DatabaseDriverSQLite3:
		return sqliteKeywords
	case DatabaseDriverMssql:
		return mssqlKeywords
	case DatabaseDriverOracle:
		return oracleKeyWords
	case DatabaseDriverH2:
		return h2Keywords
	case DatabaseDriverVertica:
		return verticaKeywords
	case DatabaseDriverClickhouse:
		return clickhouseKeywords
	default:
		return sqliteKeywords
	}
}

func DataBaseFunctions(driver DatabaseDriver) []string {
	switch driver {
	case DatabaseDriverMySQL:
		return mysql8Function
	case DatabaseDriverMySQL8:
		return mysql8Function
	case DatabaseDriverMySQL57:
		return mysql57function
	case DatabaseDriverMySQL56:
		return mysql56Function
	case DatabaseDriverPostgreSQL:
		return []string{}
	case DatabaseDriverSQLite3:
		return []string{}
	case DatabaseDriverMssql:
		return []string{}
	case DatabaseDriverOracle:
		return oracleReservedWords
	case DatabaseDriverH2:
		return []string{}
	case DatabaseDriverVertica:
		return verticaReservedWords
	case DatabaseDriverClickhouse:
		return []string{}
	default:
		return []string{}
	}
}

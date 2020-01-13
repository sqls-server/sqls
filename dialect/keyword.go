package dialect

var Keywords map[string]struct{}
var ReservedForTableAlias map[string]struct{}
var ReservedForColumnAlias map[string]struct{}

func init() {
	Keywords = make(map[string]struct{})
	Keywords[ABS] = struct{}{}
	Keywords[ADD] = struct{}{}
	Keywords[ASC] = struct{}{}
	Keywords[ALL] = struct{}{}
	Keywords[ALLOCATE] = struct{}{}
	Keywords[ALTER] = struct{}{}
	Keywords[AND] = struct{}{}
	Keywords[ANY] = struct{}{}
	Keywords[ARE] = struct{}{}
	Keywords[ARRAY] = struct{}{}
	Keywords[ARRAY_AGG] = struct{}{}
	Keywords[ARRAY_MAX_CARDINALITY] = struct{}{}
	Keywords[AS] = struct{}{}
	Keywords[ASENSITIVE] = struct{}{}
	Keywords[ASYMMETRIC] = struct{}{}
	Keywords[AT] = struct{}{}
	Keywords[ATOMIC] = struct{}{}
	Keywords[AUTHORIZATION] = struct{}{}
	Keywords[AVG] = struct{}{}
	Keywords[BEGIN] = struct{}{}
	Keywords[BEGIN_FRAME] = struct{}{}
	Keywords[BEGIN_PARTITION] = struct{}{}
	Keywords[BETWEEN] = struct{}{}
	Keywords[BIGINT] = struct{}{}
	Keywords[BINARY] = struct{}{}
	Keywords[BLOB] = struct{}{}
	Keywords[BOOLEAN] = struct{}{}
	Keywords[BOTH] = struct{}{}
	Keywords[BY] = struct{}{}
	Keywords[BYTEA] = struct{}{}
	Keywords[CALL] = struct{}{}
	Keywords[CALLED] = struct{}{}
	Keywords[CARDINALITY] = struct{}{}
	Keywords[CASCADED] = struct{}{}
	Keywords[CASE] = struct{}{}
	Keywords[CAST] = struct{}{}
	Keywords[CEIL] = struct{}{}
	Keywords[CEILING] = struct{}{}
	Keywords[CHR] = struct{}{}
	Keywords[CHAR] = struct{}{}
	Keywords[CHAR_LENGTH] = struct{}{}
	Keywords[CHARACTER] = struct{}{}
	Keywords[CHARACTER_LENGTH] = struct{}{}
	Keywords[CHECK] = struct{}{}
	Keywords[CLOB] = struct{}{}
	Keywords[CLOSE] = struct{}{}
	Keywords[COALESCE] = struct{}{}
	Keywords[COLLATE] = struct{}{}
	Keywords[COLLECT] = struct{}{}
	Keywords[COLUMN] = struct{}{}
	Keywords[COMMIT] = struct{}{}
	Keywords[CONDITION] = struct{}{}
	Keywords[CONNECT] = struct{}{}
	Keywords[CONSTRAINT] = struct{}{}
	Keywords[CONTAINS] = struct{}{}
	Keywords[CONVERT] = struct{}{}
	Keywords[COPY] = struct{}{}
	Keywords[CORR] = struct{}{}
	Keywords[CORRESPONDING] = struct{}{}
	Keywords[COUNT] = struct{}{}
	Keywords[COVAR_POP] = struct{}{}
	Keywords[COVAR_SAMP] = struct{}{}
	Keywords[CREATE] = struct{}{}
	Keywords[CROSS] = struct{}{}
	Keywords[CSV] = struct{}{}
	Keywords[CUBE] = struct{}{}
	Keywords[CUME_DIST] = struct{}{}
	Keywords[CURRENT] = struct{}{}
	Keywords[CURRENT_CATALOG] = struct{}{}
	Keywords[CURRENT_DATE] = struct{}{}
	Keywords[CURRENT_DEFAULT_TRANSFORM_GROUP] = struct{}{}
	Keywords[CURRENT_PATH] = struct{}{}
	Keywords[CURRENT_ROLE] = struct{}{}
	Keywords[CURRENT_ROW] = struct{}{}
	Keywords[CURRENT_SCHEMA] = struct{}{}
	Keywords[CURRENT_TIME] = struct{}{}
	Keywords[CURRENT_TIMESTAMP] = struct{}{}
	Keywords[CURRENT_TRANSFORM_GROUP_FOR_TYPE] = struct{}{}
	Keywords[CURRENT_USER] = struct{}{}
	Keywords[CURSOR] = struct{}{}
	Keywords[CYCLE] = struct{}{}
	Keywords[DATE] = struct{}{}
	Keywords[DAY] = struct{}{}
	Keywords[DEALLOCATE] = struct{}{}
	Keywords[DEC] = struct{}{}
	Keywords[DECIMAL] = struct{}{}
	Keywords[DECLARE] = struct{}{}
	Keywords[DEFAULT] = struct{}{}
	Keywords[DELETE] = struct{}{}
	Keywords[DENSE_RANK] = struct{}{}
	Keywords[DEREF] = struct{}{}
	Keywords[DESC] = struct{}{}
	Keywords[DESCRIBE] = struct{}{}
	Keywords[DETERMINISTIC] = struct{}{}
	Keywords[DISCONNECT] = struct{}{}
	Keywords[DISTINCT] = struct{}{}
	Keywords[DOUBLE] = struct{}{}
	Keywords[DROP] = struct{}{}
	Keywords[DYNAMIC] = struct{}{}
	Keywords[EACH] = struct{}{}
	Keywords[ELEMENT] = struct{}{}
	Keywords[ELSE] = struct{}{}
	Keywords[END] = struct{}{}
	Keywords[END_FRAME] = struct{}{}
	Keywords[END_PARTITION] = struct{}{}
	Keywords[EQUALS] = struct{}{}
	Keywords[ESCAPE] = struct{}{}
	Keywords[EVERY] = struct{}{}
	Keywords[EXCEPT] = struct{}{}
	Keywords[EXEC] = struct{}{}
	Keywords[EXECUTE] = struct{}{}
	Keywords[EXISTS] = struct{}{}
	Keywords[EXP] = struct{}{}
	Keywords[EXTERNAL] = struct{}{}
	Keywords[EXTRACT] = struct{}{}
	Keywords[FALSE] = struct{}{}
	Keywords[FETCH] = struct{}{}
	Keywords[FILTER] = struct{}{}
	Keywords[FIRST_VALUE] = struct{}{}
	Keywords[FLOAT] = struct{}{}
	Keywords[FLOOR] = struct{}{}
	Keywords[FOLLOWING] = struct{}{}
	Keywords[FOR] = struct{}{}
	Keywords[FOREIGN] = struct{}{}
	Keywords[FRAME_ROW] = struct{}{}
	Keywords[FREE] = struct{}{}
	Keywords[FROM] = struct{}{}
	Keywords[FULL] = struct{}{}
	Keywords[FUNCTION] = struct{}{}
	Keywords[FUSION] = struct{}{}
	Keywords[GET] = struct{}{}
	Keywords[GLOBAL] = struct{}{}
	Keywords[GRANT] = struct{}{}
	Keywords[GROUP] = struct{}{}
	Keywords[GROUPING] = struct{}{}
	Keywords[GROUPS] = struct{}{}
	Keywords[HAVING] = struct{}{}
	Keywords[HEADER] = struct{}{}
	Keywords[HOLD] = struct{}{}
	Keywords[HOUR] = struct{}{}
	Keywords[IDENTITY] = struct{}{}
	Keywords[IN] = struct{}{}
	Keywords[INDICATOR] = struct{}{}
	Keywords[INNER] = struct{}{}
	Keywords[INOUT] = struct{}{}
	Keywords[INSENSITIVE] = struct{}{}
	Keywords[INSERT] = struct{}{}
	Keywords[INT] = struct{}{}
	Keywords[INTEGER] = struct{}{}
	Keywords[INTERSECT] = struct{}{}
	Keywords[INTERSECTION] = struct{}{}
	Keywords[INTERVAL] = struct{}{}
	Keywords[INTO] = struct{}{}
	Keywords[IS] = struct{}{}
	Keywords[JOIN] = struct{}{}
	Keywords[KEY] = struct{}{}
	Keywords[LAG] = struct{}{}
	Keywords[LANGUAGE] = struct{}{}
	Keywords[LARGE] = struct{}{}
	Keywords[LAST_VALUE] = struct{}{}
	Keywords[LATERAL] = struct{}{}
	Keywords[LEAD] = struct{}{}
	Keywords[LEADING] = struct{}{}
	Keywords[LEFT] = struct{}{}
	Keywords[LIKE] = struct{}{}
	Keywords[LIKE_REGEX] = struct{}{}
	Keywords[LIMIT] = struct{}{}
	Keywords[LN] = struct{}{}
	Keywords[LOCAL] = struct{}{}
	Keywords[LOCALTIME] = struct{}{}
	Keywords[LOCALTIMESTAMP] = struct{}{}
	Keywords[LOCATION] = struct{}{}
	Keywords[LOWER] = struct{}{}
	Keywords[MATCH] = struct{}{}
	Keywords[MATERIALIZED] = struct{}{}
	Keywords[MAX] = struct{}{}
	Keywords[MEMBER] = struct{}{}
	Keywords[MERGE] = struct{}{}
	Keywords[METHOD] = struct{}{}
	Keywords[MIN] = struct{}{}
	Keywords[MINUTE] = struct{}{}
	Keywords[MOD] = struct{}{}
	Keywords[MODIFIES] = struct{}{}
	Keywords[MODULE] = struct{}{}
	Keywords[MONTH] = struct{}{}
	Keywords[MULTISET] = struct{}{}
	Keywords[NATIONAL] = struct{}{}
	Keywords[NATURAL] = struct{}{}
	Keywords[NCHAR] = struct{}{}
	Keywords[NCLOB] = struct{}{}
	Keywords[NEW] = struct{}{}
	Keywords[NO] = struct{}{}
	Keywords[NONE] = struct{}{}
	Keywords[NORMALIZE] = struct{}{}
	Keywords[NOT] = struct{}{}
	Keywords[NTH_VALUE] = struct{}{}
	Keywords[NTILE] = struct{}{}
	Keywords[NULL] = struct{}{}
	Keywords[NULLIF] = struct{}{}
	Keywords[NUMERIC] = struct{}{}
	Keywords[OBJECT] = struct{}{}
	Keywords[OCTET_LENGTH] = struct{}{}
	Keywords[OCCURRENCES_REGEX] = struct{}{}
	Keywords[OF] = struct{}{}
	Keywords[OFFSET] = struct{}{}
	Keywords[OLD] = struct{}{}
	Keywords[ON] = struct{}{}
	Keywords[ONLY] = struct{}{}
	Keywords[OPEN] = struct{}{}
	Keywords[OR] = struct{}{}
	Keywords[ORDER] = struct{}{}
	Keywords[OUT] = struct{}{}
	Keywords[OUTER] = struct{}{}
	Keywords[OVER] = struct{}{}
	Keywords[OVERLAPS] = struct{}{}
	Keywords[OVERLAY] = struct{}{}
	Keywords[PARAMETER] = struct{}{}
	Keywords[PARTITION] = struct{}{}
	Keywords[PARQUET] = struct{}{}
	Keywords[PERCENT] = struct{}{}
	Keywords[PERCENT_RANK] = struct{}{}
	Keywords[PERCENTILE_CONT] = struct{}{}
	Keywords[PERCENTILE_DISC] = struct{}{}
	Keywords[PERIOD] = struct{}{}
	Keywords[PORTION] = struct{}{}
	Keywords[POSITION] = struct{}{}
	Keywords[POSITION_REGEX] = struct{}{}
	Keywords[POWER] = struct{}{}
	Keywords[PRECEDES] = struct{}{}
	Keywords[PRECEDING] = struct{}{}
	Keywords[PRECISION] = struct{}{}
	Keywords[PREPARE] = struct{}{}
	Keywords[PRIMARY] = struct{}{}
	Keywords[PROCEDURE] = struct{}{}
	Keywords[RANGE] = struct{}{}
	Keywords[RANK] = struct{}{}
	Keywords[READS] = struct{}{}
	Keywords[REAL] = struct{}{}
	Keywords[RECURSIVE] = struct{}{}
	Keywords[REF] = struct{}{}
	Keywords[REFERENCES] = struct{}{}
	Keywords[REFERENCING] = struct{}{}
	Keywords[REGCLASS] = struct{}{}
	Keywords[REGR_AVGX] = struct{}{}
	Keywords[REGR_AVGY] = struct{}{}
	Keywords[REGR_COUNT] = struct{}{}
	Keywords[REGR_INTERCEPT] = struct{}{}
	Keywords[REGR_R2] = struct{}{}
	Keywords[REGR_SLOPE] = struct{}{}
	Keywords[REGR_SXX] = struct{}{}
	Keywords[REGR_SXY] = struct{}{}
	Keywords[REGR_SYY] = struct{}{}
	Keywords[RELEASE] = struct{}{}
	Keywords[RESULT] = struct{}{}
	Keywords[RETURN] = struct{}{}
	Keywords[RETURNS] = struct{}{}
	Keywords[REVOKE] = struct{}{}
	Keywords[RIGHT] = struct{}{}
	Keywords[ROLLBACK] = struct{}{}
	Keywords[ROLLUP] = struct{}{}
	Keywords[ROW] = struct{}{}
	Keywords[ROW_NUMBER] = struct{}{}
	Keywords[ROWS] = struct{}{}
	Keywords[SAVEPOINT] = struct{}{}
	Keywords[SCOPE] = struct{}{}
	Keywords[SCROLL] = struct{}{}
	Keywords[SEARCH] = struct{}{}
	Keywords[SECOND] = struct{}{}
	Keywords[SELECT] = struct{}{}
	Keywords[SENSITIVE] = struct{}{}
	Keywords[SESSION_USER] = struct{}{}
	Keywords[SET] = struct{}{}
	Keywords[SIMILAR] = struct{}{}
	Keywords[SMALLINT] = struct{}{}
	Keywords[SOME] = struct{}{}
	Keywords[SPECIFIC] = struct{}{}
	Keywords[SPECIFICTYPE] = struct{}{}
	Keywords[SQL] = struct{}{}
	Keywords[SQLEXCEPTION] = struct{}{}
	Keywords[SQLSTATE] = struct{}{}
	Keywords[SQLWARNING] = struct{}{}
	Keywords[SQRT] = struct{}{}
	Keywords[START] = struct{}{}
	Keywords[STATIC] = struct{}{}
	Keywords[STDDEV_POP] = struct{}{}
	Keywords[STDDEV_SAMP] = struct{}{}
	Keywords[STDIN] = struct{}{}
	Keywords[STORED] = struct{}{}
	Keywords[SUBMULTISET] = struct{}{}
	Keywords[SUBSTRING] = struct{}{}
	Keywords[SUBSTRING_REGEX] = struct{}{}
	Keywords[SUCCEEDS] = struct{}{}
	Keywords[SUM] = struct{}{}
	Keywords[SYMMETRIC] = struct{}{}
	Keywords[SYSTEM] = struct{}{}
	Keywords[SYSTEM_TIME] = struct{}{}
	Keywords[SYSTEM_USER] = struct{}{}
	Keywords[TABLE] = struct{}{}
	Keywords[TABLESAMPLE] = struct{}{}
	Keywords[TEXT] = struct{}{}
	Keywords[THEN] = struct{}{}
	Keywords[TIME] = struct{}{}
	Keywords[TIMESTAMP] = struct{}{}
	Keywords[TIMEZONE_HOUR] = struct{}{}
	Keywords[TIMEZONE_MINUTE] = struct{}{}
	Keywords[TO] = struct{}{}
	Keywords[TRAILING] = struct{}{}
	Keywords[TRANSLATE] = struct{}{}
	Keywords[TRANSLATE_REGEX] = struct{}{}
	Keywords[TRANSLATION] = struct{}{}
	Keywords[TREAT] = struct{}{}
	Keywords[TRIGGER] = struct{}{}
	Keywords[TRUNCATE] = struct{}{}
	Keywords[TRIM] = struct{}{}
	Keywords[TRIM_ARRAY] = struct{}{}
	Keywords[TRUE] = struct{}{}
	Keywords[UESCAPE] = struct{}{}
	Keywords[UNBOUNDED] = struct{}{}
	Keywords[UNION] = struct{}{}
	Keywords[UNIQUE] = struct{}{}
	Keywords[UNKNOWN] = struct{}{}
	Keywords[UNNEST] = struct{}{}
	Keywords[UPDATE] = struct{}{}
	Keywords[UPPER] = struct{}{}
	Keywords[USER] = struct{}{}
	Keywords[USING] = struct{}{}
	Keywords[UUID] = struct{}{}
	Keywords[VALUE] = struct{}{}
	Keywords[VALUES] = struct{}{}
	Keywords[VALUE_OF] = struct{}{}
	Keywords[VAR_POP] = struct{}{}
	Keywords[VAR_SAMP] = struct{}{}
	Keywords[VARBINARY] = struct{}{}
	Keywords[VARCHAR] = struct{}{}
	Keywords[VARYING] = struct{}{}
	Keywords[VERSIONING] = struct{}{}
	Keywords[VIEW] = struct{}{}
	Keywords[WHEN] = struct{}{}
	Keywords[WHENEVER] = struct{}{}
	Keywords[WHERE] = struct{}{}
	Keywords[WIDTH_BUCKET] = struct{}{}
	Keywords[WINDOW] = struct{}{}
	Keywords[WITH] = struct{}{}
	Keywords[WITHIN] = struct{}{}
	Keywords[WITHOUT] = struct{}{}
	Keywords[YEAR] = struct{}{}
	Keywords[ZONE] = struct{}{}

	ReservedForTableAlias = make(map[string]struct{})
	ReservedForTableAlias[WITH] = struct{}{}
	ReservedForTableAlias[SELECT] = struct{}{}
	ReservedForTableAlias[WHERE] = struct{}{}
	ReservedForTableAlias[GROUP] = struct{}{}
	ReservedForTableAlias[ORDER] = struct{}{}
	ReservedForTableAlias[UNION] = struct{}{}
	ReservedForTableAlias[EXCEPT] = struct{}{}
	ReservedForTableAlias[INTERSECT] = struct{}{}
	ReservedForTableAlias[ON] = struct{}{}
	ReservedForTableAlias[JOIN] = struct{}{}
	ReservedForTableAlias[INNER] = struct{}{}
	ReservedForTableAlias[CROSS] = struct{}{}
	ReservedForTableAlias[FULL] = struct{}{}
	ReservedForTableAlias[LEFT] = struct{}{}
	ReservedForTableAlias[RIGHT] = struct{}{}
	ReservedForTableAlias[NATURAL] = struct{}{}
	ReservedForTableAlias[USING] = struct{}{}

	ReservedForColumnAlias = make(map[string]struct{})
	ReservedForColumnAlias[WITH] = struct{}{}
	ReservedForColumnAlias[SELECT] = struct{}{}
	ReservedForColumnAlias[WHERE] = struct{}{}
	ReservedForColumnAlias[GROUP] = struct{}{}
	ReservedForColumnAlias[ORDER] = struct{}{}
	ReservedForColumnAlias[UNION] = struct{}{}
	ReservedForColumnAlias[EXCEPT] = struct{}{}
	ReservedForColumnAlias[INTERSECT] = struct{}{}
	ReservedForColumnAlias[FROM] = struct{}{}
}

const (
	ABS                              string = "ABS"
	ADD                                     = "ADD"
	ASC                                     = "ASC"
	ALL                                     = "ALL"
	ALLOCATE                                = "ALLOCATE"
	ALTER                                   = "ALTER"
	AND                                     = "AND"
	ANY                                     = "ANY"
	ARE                                     = "ARE"
	ARRAY                                   = "ARRAY"
	ARRAY_AGG                               = "ARRAY_AGG"
	ARRAY_MAX_CARDINALITY                   = "ARRAY_MAX_CARDINALITY"
	AS                                      = "AS"
	ASENSITIVE                              = "ASENSITIVE"
	ASYMMETRIC                              = "ASYMMETRIC"
	AT                                      = "AT"
	ATOMIC                                  = "ATOMIC"
	AUTHORIZATION                           = "AUTHORIZATION"
	AVG                                     = "AVG"
	BEGIN                                   = "BEGIN"
	BEGIN_FRAME                             = "BEGIN_FRAME"
	BEGIN_PARTITION                         = "BEGIN_PARTITION"
	BETWEEN                                 = "BETWEEN"
	BIGINT                                  = "BIGINT"
	BINARY                                  = "BINARY"
	BLOB                                    = "BLOB"
	BOOLEAN                                 = "BOOLEAN"
	BOTH                                    = "BOTH"
	BY                                      = "BY"
	BYTEA                                   = "BYTEA"
	CALL                                    = "CALL"
	CALLED                                  = "CALLED"
	CARDINALITY                             = "CARDINALITY"
	CASCADED                                = "CASCADED"
	CASE                                    = "CASE"
	CAST                                    = "CAST"
	CEIL                                    = "CEIL"
	CEILING                                 = "CEILING"
	CHR                                     = "CHR"
	CHAR                                    = "CHAR"
	CHAR_LENGTH                             = "CHAR_LENGTH"
	CHARACTER                               = "CHARACTER"
	CHARACTER_LENGTH                        = "CHARACTER_LENGTH"
	CHECK                                   = "CHECK"
	CLOB                                    = "CLOB"
	CLOSE                                   = "CLOSE"
	COALESCE                                = "COALESCE"
	COLLATE                                 = "COLLATE"
	COLLECT                                 = "COLLECT"
	COLUMN                                  = "COLUMN"
	COMMIT                                  = "COMMIT"
	CONDITION                               = "CONDITION"
	CONNECT                                 = "CONNECT"
	CONSTRAINT                              = "CONSTRAINT"
	CONTAINS                                = "CONTAINS"
	CONVERT                                 = "CONVERT"
	COPY                                    = "COPY"
	CORR                                    = "CORR"
	CORRESPONDING                           = "CORRESPONDING"
	COUNT                                   = "COUNT"
	COVAR_POP                               = "COVAR_POP"
	COVAR_SAMP                              = "COVAR_SAMP"
	CREATE                                  = "CREATE"
	CROSS                                   = "CROSS"
	CSV                                     = "CSV"
	CUBE                                    = "CUBE"
	CUME_DIST                               = "CUME_DIST"
	CURRENT                                 = "CURRENT"
	CURRENT_CATALOG                         = "CURRENT_CATALOG"
	CURRENT_DATE                            = "CURRENT_DATE"
	CURRENT_DEFAULT_TRANSFORM_GROUP         = "CURRENT_DEFAULT_TRANSFORM_GROUP"
	CURRENT_PATH                            = "CURRENT_PATH"
	CURRENT_ROLE                            = "CURRENT_ROLE"
	CURRENT_ROW                             = "CURRENT_ROW"
	CURRENT_SCHEMA                          = "CURRENT_SCHEMA"
	CURRENT_TIME                            = "CURRENT_TIME"
	CURRENT_TIMESTAMP                       = "CURRENT_TIMESTAMP"
	CURRENT_TRANSFORM_GROUP_FOR_TYPE        = "CURRENT_TRANSFORM_GROUP_FOR_TYPE"
	CURRENT_USER                            = "CURRENT_USER"
	CURSOR                                  = "CURSOR"
	CYCLE                                   = "CYCLE"
	DATE                                    = "DATE"
	DAY                                     = "DAY"
	DEALLOCATE                              = "DEALLOCATE"
	DEC                                     = "DEC"
	DECIMAL                                 = "DECIMAL"
	DECLARE                                 = "DECLARE"
	DEFAULT                                 = "DEFAULT"
	DELETE                                  = "DELETE"
	DENSE_RANK                              = "DENSE_RANK"
	DEREF                                   = "DEREF"
	DESC                                    = "DESC"
	DESCRIBE                                = "DESCRIBE"
	DETERMINISTIC                           = "DETERMINISTIC"
	DISCONNECT                              = "DISCONNECT"
	DISTINCT                                = "DISTINCT"
	DOUBLE                                  = "DOUBLE"
	DROP                                    = "DROP"
	DYNAMIC                                 = "DYNAMIC"
	EACH                                    = "EACH"
	ELEMENT                                 = "ELEMENT"
	ELSE                                    = "ELSE"
	END                                     = "END"
	END_FRAME                               = "END_FRAME"
	END_PARTITION                           = "END_PARTITION"
	EQUALS                                  = "EQUALS"
	ESCAPE                                  = "ESCAPE"
	EVERY                                   = "EVERY"
	EXCEPT                                  = "EXCEPT"
	EXEC                                    = "EXEC"
	EXECUTE                                 = "EXECUTE"
	EXISTS                                  = "EXISTS"
	EXP                                     = "EXP"
	EXTERNAL                                = "EXTERNAL"
	EXTRACT                                 = "EXTRACT"
	FALSE                                   = "FALSE"
	FETCH                                   = "FETCH"
	FILTER                                  = "FILTER"
	FIRST_VALUE                             = "FIRST_VALUE"
	FLOAT                                   = "FLOAT"
	FLOOR                                   = "FLOOR"
	FOLLOWING                               = "FOLLOWING"
	FOR                                     = "FOR"
	FOREIGN                                 = "FOREIGN"
	FRAME_ROW                               = "FRAME_ROW"
	FREE                                    = "FREE"
	FROM                                    = "FROM"
	FULL                                    = "FULL"
	FUNCTION                                = "FUNCTION"
	FUSION                                  = "FUSION"
	GET                                     = "GET"
	GLOBAL                                  = "GLOBAL"
	GRANT                                   = "GRANT"
	GROUP                                   = "GROUP"
	GROUPING                                = "GROUPING"
	GROUPS                                  = "GROUPS"
	HAVING                                  = "HAVING"
	HEADER                                  = "HEADER"
	HOLD                                    = "HOLD"
	HOUR                                    = "HOUR"
	IDENTITY                                = "IDENTITY"
	IN                                      = "IN"
	INDICATOR                               = "INDICATOR"
	INNER                                   = "INNER"
	INOUT                                   = "INOUT"
	INSENSITIVE                             = "INSENSITIVE"
	INSERT                                  = "INSERT"
	INT                                     = "INT"
	INTEGER                                 = "INTEGER"
	INTERSECT                               = "INTERSECT"
	INTERSECTION                            = "INTERSECTION"
	INTERVAL                                = "INTERVAL"
	INTO                                    = "INTO"
	IS                                      = "IS"
	JOIN                                    = "JOIN"
	KEY                                     = "KEY"
	LAG                                     = "LAG"
	LANGUAGE                                = "LANGUAGE"
	LARGE                                   = "LARGE"
	LAST_VALUE                              = "LAST_VALUE"
	LATERAL                                 = "LATERAL"
	LEAD                                    = "LEAD"
	LEADING                                 = "LEADING"
	LEFT                                    = "LEFT"
	LIKE                                    = "LIKE"
	LIKE_REGEX                              = "LIKE_REGEX"
	LIMIT                                   = "LIMIT"
	LN                                      = "LN"
	LOCAL                                   = "LOCAL"
	LOCALTIME                               = "LOCALTIME"
	LOCALTIMESTAMP                          = "LOCALTIMESTAMP"
	LOCATION                                = "LOCATION"
	LOWER                                   = "LOWER"
	MATCH                                   = "MATCH"
	MATERIALIZED                            = "MATERIALIZED"
	MAX                                     = "MAX"
	MEMBER                                  = "MEMBER"
	MERGE                                   = "MERGE"
	METHOD                                  = "METHOD"
	MIN                                     = "MIN"
	MINUTE                                  = "MINUTE"
	MOD                                     = "MOD"
	MODIFIES                                = "MODIFIES"
	MODULE                                  = "MODULE"
	MONTH                                   = "MONTH"
	MULTISET                                = "MULTISET"
	NATIONAL                                = "NATIONAL"
	NATURAL                                 = "NATURAL"
	NCHAR                                   = "NCHAR"
	NCLOB                                   = "NCLOB"
	NEW                                     = "NEW"
	NO                                      = "NO"
	NONE                                    = "NONE"
	NORMALIZE                               = "NORMALIZE"
	NOT                                     = "NOT"
	NTH_VALUE                               = "NTH_VALUE"
	NTILE                                   = "NTILE"
	NULL                                    = "NULL"
	NULLIF                                  = "NULLIF"
	NUMERIC                                 = "NUMERIC"
	OBJECT                                  = "OBJECT"
	OCTET_LENGTH                            = "OCTET_LENGTH"
	OCCURRENCES_REGEX                       = "OCCURRENCES_REGEX"
	OF                                      = "OF"
	OFFSET                                  = "OFFSET"
	OLD                                     = "OLD"
	ON                                      = "ON"
	ONLY                                    = "ONLY"
	OPEN                                    = "OPEN"
	OR                                      = "OR"
	ORDER                                   = "ORDER"
	OUT                                     = "OUT"
	OUTER                                   = "OUTER"
	OVER                                    = "OVER"
	OVERLAPS                                = "OVERLAPS"
	OVERLAY                                 = "OVERLAY"
	PARAMETER                               = "PARAMETER"
	PARTITION                               = "PARTITION"
	PARQUET                                 = "PARQUET"
	PERCENT                                 = "PERCENT"
	PERCENT_RANK                            = "PERCENT_RANK"
	PERCENTILE_CONT                         = "PERCENTILE_CONT"
	PERCENTILE_DISC                         = "PERCENTILE_DISC"
	PERIOD                                  = "PERIOD"
	PORTION                                 = "PORTION"
	POSITION                                = "POSITION"
	POSITION_REGEX                          = "POSITION_REGEX"
	POWER                                   = "POWER"
	PRECEDES                                = "PRECEDES"
	PRECEDING                               = "PRECEDING"
	PRECISION                               = "PRECISION"
	PREPARE                                 = "PREPARE"
	PRIMARY                                 = "PRIMARY"
	PROCEDURE                               = "PROCEDURE"
	RANGE                                   = "RANGE"
	RANK                                    = "RANK"
	READS                                   = "READS"
	REAL                                    = "REAL"
	RECURSIVE                               = "RECURSIVE"
	REF                                     = "REF"
	REFERENCES                              = "REFERENCES"
	REFERENCING                             = "REFERENCING"
	REGCLASS                                = "REGCLASS"
	REGR_AVGX                               = "REGR_AVGX"
	REGR_AVGY                               = "REGR_AVGY"
	REGR_COUNT                              = "REGR_COUNT"
	REGR_INTERCEPT                          = "REGR_INTERCEPT"
	REGR_R2                                 = "REGR_R2"
	REGR_SLOPE                              = "REGR_SLOPE"
	REGR_SXX                                = "REGR_SXX"
	REGR_SXY                                = "REGR_SXY"
	REGR_SYY                                = "REGR_SYY"
	RELEASE                                 = "RELEASE"
	RESULT                                  = "RESULT"
	RETURN                                  = "RETURN"
	RETURNS                                 = "RETURNS"
	REVOKE                                  = "REVOKE"
	RIGHT                                   = "RIGHT"
	ROLLBACK                                = "ROLLBACK"
	ROLLUP                                  = "ROLLUP"
	ROW                                     = "ROW"
	ROW_NUMBER                              = "ROW_NUMBER"
	ROWS                                    = "ROWS"
	SAVEPOINT                               = "SAVEPOINT"
	SCOPE                                   = "SCOPE"
	SCROLL                                  = "SCROLL"
	SEARCH                                  = "SEARCH"
	SECOND                                  = "SECOND"
	SELECT                                  = "SELECT"
	SENSITIVE                               = "SENSITIVE"
	SESSION_USER                            = "SESSION_USER"
	SET                                     = "SET"
	SIMILAR                                 = "SIMILAR"
	SMALLINT                                = "SMALLINT"
	SOME                                    = "SOME"
	SPECIFIC                                = "SPECIFIC"
	SPECIFICTYPE                            = "SPECIFICTYPE"
	SQL                                     = "SQL"
	SQLEXCEPTION                            = "SQLEXCEPTION"
	SQLSTATE                                = "SQLSTATE"
	SQLWARNING                              = "SQLWARNING"
	SQRT                                    = "SQRT"
	START                                   = "START"
	STATIC                                  = "STATIC"
	STDDEV_POP                              = "STDDEV_POP"
	STDDEV_SAMP                             = "STDDEV_SAMP"
	STDIN                                   = "STDIN"
	STORED                                  = "STORED"
	SUBMULTISET                             = "SUBMULTISET"
	SUBSTRING                               = "SUBSTRING"
	SUBSTRING_REGEX                         = "SUBSTRING_REGEX"
	SUCCEEDS                                = "SUCCEEDS"
	SUM                                     = "SUM"
	SYMMETRIC                               = "SYMMETRIC"
	SYSTEM                                  = "SYSTEM"
	SYSTEM_TIME                             = "SYSTEM_TIME"
	SYSTEM_USER                             = "SYSTEM_USER"
	TABLE                                   = "TABLE"
	TABLESAMPLE                             = "TABLESAMPLE"
	TEXT                                    = "TEXT"
	THEN                                    = "THEN"
	TIME                                    = "TIME"
	TIMESTAMP                               = "TIMESTAMP"
	TIMEZONE_HOUR                           = "TIMEZONE_HOUR"
	TIMEZONE_MINUTE                         = "TIMEZONE_MINUTE"
	TO                                      = "TO"
	TRAILING                                = "TRAILING"
	TRANSLATE                               = "TRANSLATE"
	TRANSLATE_REGEX                         = "TRANSLATE_REGEX"
	TRANSLATION                             = "TRANSLATION"
	TREAT                                   = "TREAT"
	TRIGGER                                 = "TRIGGER"
	TRUNCATE                                = "TRUNCATE"
	TRIM                                    = "TRIM"
	TRIM_ARRAY                              = "TRIM_ARRAY"
	TRUE                                    = "TRUE"
	UESCAPE                                 = "UESCAPE"
	UNBOUNDED                               = "UNBOUNDED"
	UNION                                   = "UNION"
	UNIQUE                                  = "UNIQUE"
	UNKNOWN                                 = "UNKNOWN"
	UNNEST                                  = "UNNEST"
	UPDATE                                  = "UPDATE"
	UPPER                                   = "UPPER"
	USER                                    = "USER"
	USING                                   = "USING"
	UUID                                    = "UUID"
	VALUE                                   = "VALUE"
	VALUES                                  = "VALUES"
	VALUE_OF                                = "VALUE_OF"
	VAR_POP                                 = "VAR_POP"
	VAR_SAMP                                = "VAR_SAMP"
	VARBINARY                               = "VARBINARY"
	VARCHAR                                 = "VARCHAR"
	VARYING                                 = "VARYING"
	VERSIONING                              = "VERSIONING"
	VIEW                                    = "VIEW"
	WHEN                                    = "WHEN"
	WHENEVER                                = "WHENEVER"
	WHERE                                   = "WHERE"
	WIDTH_BUCKET                            = "WIDTH_BUCKET"
	WINDOW                                  = "WINDOW"
	WITH                                    = "WITH"
	WITHIN                                  = "WITHIN"
	WITHOUT                                 = "WITHOUT"
	YEAR                                    = "YEAR"
	ZONE                                    = "ZONE"
)

type KeywordKind int

//go:generate stringer -type KeywordKind kind.go
const (
	// Unmatched keyword (like table and column identifier)
	Unmatched KeywordKind = iota
	// Matched keyword
	Matched
	// Data Manipulation Language
	DML
	// Data Definition Language
	DDL
	// Data Control Language
	DCL
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

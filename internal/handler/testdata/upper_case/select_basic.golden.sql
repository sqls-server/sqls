SELECT
	a,
	b AS bb,
	c
FROM
	tbl
JOIN (
	SELECT
		a * 2 AS a
	FROM
		new_table
) other
	ON tbl.a = other.a
WHERE
	c IS TRUE
	AND b BETWEEN 3
	AND 4
	OR d IS 'blue'
LIMIT 10
select
	a,
	case
		when a = 0 then 1
		when bb = 1 then 1
		when c = 2 then 2
		else 0
	end as d,
	extra_col
from
	table
where
	c is true
	and b between 3
	and 4
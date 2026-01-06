select
	a,
	b,
	c,
	sum(x) as sum_x,
	count(y) as cnt_y
from
	table
group by
	a,
	b,
	c
having
	sum(x) > 1
	and count(y) > 5
order by
	3,
	2,
	1
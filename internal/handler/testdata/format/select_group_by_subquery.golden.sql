select
	*,
	sum_b + 2 as mod_sum
from
	(
		select
			a,
			sum(b) as sum_b
		from
			table
		group by
			a,
			z
	)
order by
	1,
	2
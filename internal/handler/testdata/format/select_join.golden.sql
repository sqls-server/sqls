select
	*
from
	a
join b
	on a.one = b.one
left join c
	on c.two = a.two
	and c.three = a.three
right outer join d
	on d.three = a.three
cross join e
	on e.four = a.four
join f using (
	one,
	two,
	three
)
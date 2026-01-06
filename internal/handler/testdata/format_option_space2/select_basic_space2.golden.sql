select
  a,
  b as bb,
  c
from
  table
join (
  select
    a * 2 as a
  from
    new_table
) other
  on table.a = other.a
where
  c is true
  and b between 3
  and 4
  or d is 'blue'
limit 10
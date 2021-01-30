select a, b as bb,c from tbl
join (select a * 2 as a from new_table) other
on tbl.a = other.a
where c is true
and b between 3 and 4
or d is 'blue'
limit 10

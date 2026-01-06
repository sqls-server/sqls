update users
set email = 'new@example.com'
where
	id = 1;
delete from users
where
	id = 1;
update users
set age = age + 1
where
	status = 'active';

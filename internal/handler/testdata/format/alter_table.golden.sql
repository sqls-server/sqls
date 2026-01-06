alter table users add column age int;
alter table users drop column age;
alter table users MODIFY column email varchar(255) not null;
alter table orders add constraint fk_user foreign key(user_id) references users(id);

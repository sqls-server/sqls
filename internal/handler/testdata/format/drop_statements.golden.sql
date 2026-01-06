drop table if exists users;
drop INDEX idx_users_email
on users;
drop trigger if exists update_timestamp;

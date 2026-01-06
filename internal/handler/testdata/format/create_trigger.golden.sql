create trigger update_timestamp BEFOREupdate
	on users for each row begin
set new.updated_at = NOW();

end;

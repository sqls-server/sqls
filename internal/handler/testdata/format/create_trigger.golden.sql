create trigger update_timestamp BEFORE update 
on users for each row begin
set new.updated_at = NOW();

end;

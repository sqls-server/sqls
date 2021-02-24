## Start test databases

```sh
docker-compose up -d
```

## MySQL setup

```sh
wget https://downloads.mysql.com/docs/world.sql.gz
gzip world.sql.gz

# MySQL 5.6
mysql -u root -proot -h 127.0.0.1 -P 13305 < world.sql
# MySQL 5.7
mysql -u root -proot -h 127.0.0.1 -P 13306 < world.sql
# MySQL 8
mysql -u root -proot -h 127.0.0.1 -P 13307 < world.sql
rm world.sql
```

## Export keyword & function list

```sh
mysql -u root -proot -h 127.0.0.1 -P 13305 -D mysql < help_categories.sql > ./export/help_categories_mysql56.txt
mysql -u root -proot -h 127.0.0.1 -P 13306 -D mysql < help_categories.sql > ./export/help_categories_mysql57.txt
mysql -u root -proot -h 127.0.0.1 -P 13307 -D mysql < help_categories.sql > ./export/help_categories_mysql8.txt
# Export keyword list
mysql -u root -proot -h 127.0.0.1 -P 13305 -D mysql < help_keywords_mysql56.sql > ./export/help_keywords_mysql56.txt
mysql -u root -proot -h 127.0.0.1 -P 13306 -D mysql < help_keywords_mysql57.sql > ./export/help_keywords_mysql57.txt
mysql -u root -proot -h 127.0.0.1 -P 13307 -D mysql < help_keywords_mysql8.sql  > ./export/help_keywords_mysql8.txt
# Export function list
mysql -u root -proot -h 127.0.0.1 -P 13305 -D mysql < help_functions_mysql56.sql > ./export/help_functions_mysql56.txt
mysql -u root -proot -h 127.0.0.1 -P 13306 -D mysql < help_functions_mysql57.sql > ./export/help_functions_mysql57.txt
mysql -u root -proot -h 127.0.0.1 -P 13307 -D mysql < help_functions_mysql8.sql  > ./export/help_functions_mysql8.txt
```

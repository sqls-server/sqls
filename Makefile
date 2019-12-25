.PHONY: up
up:
	docker-compose up -d

.PHONY: mysql
up:
	wget https://downloads.mysql.com/docs/world.sql.gz
	gzip -d ./world.sql.gz
	cp ./world.sql ./docker/mysql/sql

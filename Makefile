postgres:
	sudo docker run --name postgres17 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=123456 -d postgres:17.5-alpine3.22

createdb: 
	sudo docker exec -it postgres17 createdb --username=root --owner=root ticket

dropdb: 
	sudo docker exec -it postgres17 dropdb ticket

psql: 
	sudo docker exec -it postgres17 psql -U root -d ticket

test:
	go test -v -cover ./...

run:
	go run main.go

.PHONY: postgres createdb dropdb init destroy psql test run
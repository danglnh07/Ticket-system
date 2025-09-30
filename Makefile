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

ngrok:
	sudo docker run --net=host -it -e NGROK_AUTHTOKEN=32PXyfTUHPC79zy78HEwkxIWaF0_4vevWEuVhtT8XSfuswciM ngrok/ngrok:latest http --url=kimberlie-millesimal-muscly.ngrok-free.dev 8080

.PHONY: postgres createdb dropdb init destroy psql test run ngrok

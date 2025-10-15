b:
	go build -o bin/rh ./cmd/app/main.go
u:
	docker-compose up --build
d:
	docker-compose down -v
a:
	docker exec -it ridehail-postgres psql -U ridehail_user -W ridehail_db

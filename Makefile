create-migration:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

migrate-up:
	migrate -database postgres://postgres:password@localhost:5432/seahorsefi?sslmode=disable -path ./migrations up

migrate-down:
	migrate -database postgres://postgres:password@localhost:5432/seahorsefi?sslmode=disable -path ./migrations down

up-db:
	docker run --name seahorsefi-db \
		-e POSTGRES_DB=seahorsefi \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=password \
		-p 5432:5432 \
		-d postgres

# go install github.com/swaggo/swag/cmd/swag@latest
gen-doc:
	swag init --parseInternal

run:
	go run main.go
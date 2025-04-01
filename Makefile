.PHONY:

build: 
	go build -o ./bot cmd/bot/main.go

run: build 
	./bot


compose:
	sudo docker compose up -d

compose-down:
	sudo docker compose down

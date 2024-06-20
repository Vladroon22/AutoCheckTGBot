.PHONY:

build: 
	go build -o ./bot cmd/bot/main.go

run: build 
	./bot

image: 
	sudo docker build . -t tgbot:1.0
image-rm:
	sudo docker rmi tgbot:1.0

docker:
	sudo docker run --name=tgbot -d tgbot:1.0
docker-rm:
	sudo docker stop tgbot:1.0
	sudo docker rm tgbot:1.0
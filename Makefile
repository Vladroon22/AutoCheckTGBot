.PHONY:

build: 
	go build -o ./bot cmd/bot/main.go

run: build 
	./bot

image: 
	sudo docker build . -t tgbot
image-rm:
	sudo docker rmi tgbot

docker:
	sudo docker run --name=tgbot -d tgbot
docker-rm:
	sudo docker stop tgbot
	sudo docker rm tgbot

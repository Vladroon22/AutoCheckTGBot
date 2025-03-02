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
	sudo docker network create my-network
	sudo docker run --name=tgbot --network my-network -d tgbot
	sudo docker run --name=my-mongo -p 27017:27017 --network my-network -d mongo:8.0

docker-rm:
	sudo docker rm -f tgbot
	sudo docker rm -f my-mongo
	sudo docker network rm my-network

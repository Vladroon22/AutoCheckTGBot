FROM golang:1.22.1

WORKDIR /TGBOT

RUN apt-get update && apt-get install -y make

COPY . /TGBOT/

RUN go mod download
RUN make 

CMD [ "./bot" ]
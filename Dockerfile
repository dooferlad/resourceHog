# syntax=docker/dockerfile:1

FROM golang:1.19

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download && go mod verify

COPY *.go ./

RUN go build -v -o /usr/local/bin/app ./...

EXPOSE 6776

CMD [ "app" ]
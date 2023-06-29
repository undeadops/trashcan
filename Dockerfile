# syntax=docker/dockerfile:latest
FROM golang:1.20-alpine as build-base

WORKDIR /app

RUN apk --no-cache add git

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o trashcan *.go

FROM alpine
WORKDIR /app

COPY --from=build-base /app/trashcan /app

EXPOSE 3000 9090
CMD ["/app/trashcan"]
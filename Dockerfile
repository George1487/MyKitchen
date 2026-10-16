FROM golang:1.26 AS base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

FROM base AS restaurant-demo

RUN go build -o restaurant-demo ./cmd/restaurant-demo

CMD ["./restaurant-demo"]

FROM base AS app

RUN go build -o app ./cmd/app

EXPOSE 8080 9090

CMD ["./app"]

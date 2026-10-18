FROM golang:1.26 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN mkdir -p /out \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/restaurant-demo ./cmd/restaurant-demo \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/provision-partner ./cmd/provision-partner

FROM scratch AS restaurant-demo

COPY --from=builder /out/restaurant-demo /restaurant-demo

CMD ["/restaurant-demo"]

FROM scratch AS app

COPY --from=builder /out/app /app

EXPOSE 8080 9090

CMD ["/app"]

FROM scratch AS provision-partner

COPY --from=builder /out/provision-partner /provision-partner

ENTRYPOINT ["/provision-partner"]

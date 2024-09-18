# syntax=docker/dockerfile:1

FROM golang:1.22 as builder

WORKDIR /app/

COPY go.* ./

RUN go mod download

COPY *.go ./

RUN go build -o /build

FROM golang:1.22

WORKDIR /app

ENV APP_ENV=PRODUCTION

COPY --from=builder /build .

ENV ADDRESS=:http

EXPOSE 80

CMD ["/app/build"]
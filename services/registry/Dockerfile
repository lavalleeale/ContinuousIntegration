# syntax=docker/dockerfile:1

FROM golang:1.19 as builder

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /registry-auth

FROM alpine:latest

WORKDIR /app

COPY --from=builder /registry-auth /registry-auth

EXPOSE 8081

CMD ["/registry-auth"]
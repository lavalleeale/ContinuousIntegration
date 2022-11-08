FROM node:16 as assets
WORKDIR /app
COPY package.json .
COPY yarn.lock .
RUN yarn install --frozen-lockfile
COPY . .
RUN yarn build
FROM golang:1.19 as builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
COPY --from=assets /app/assets/output ./assets/output
RUN go build -o /build
FROM golang:1.19 as runner
WORKDIR /app
ENV APP_ENV=PRODUCTION
COPY --from=builder /build .
EXPOSE 8080
CMD ["/app/build"]

FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod main.go ./
RUN go build -o /test-provider .

FROM alpine:3.20
COPY --from=builder /test-provider /test-provider

EXPOSE 5556
ENTRYPOINT ["/test-provider"]

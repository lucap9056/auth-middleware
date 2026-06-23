FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY database/ ./database/
COPY jwt/ ./jwt/
COPY oauth2/ ./oauth2/

WORKDIR /app/oauth2

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -v -ldflags "-X main.mode=production" -o /dist/oauth2 ./cmd


FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /dist/oauth2 .

EXPOSE 80

ENTRYPOINT ["./oauth2"]

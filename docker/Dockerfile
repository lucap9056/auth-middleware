FROM golang:1.27-alpine AS builder

WORKDIR /src

COPY oauth2/go.mod oauth2/go.sum oauth2/
COPY server/go.mod server/go.sum server/
RUN cd server && go mod download
COPY oauth2/ oauth2/
COPY server/ server/

RUN cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -v -o /dist/corvauth ./cmd


FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /dist/corvauth .

EXPOSE 80

ENTRYPOINT ["./corvauth"]

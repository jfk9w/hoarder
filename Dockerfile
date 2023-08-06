FROM golang:alpine AS builder
WORKDIR /src
ADD . .
RUN apk add --no-cache git
RUN go build -o /app /src/cmd/hoarder

FROM alpine:latest
COPY --from=builder /app /usr/bin/app
RUN apk add --no-cache tzdata
ENTRYPOINT ["app"]

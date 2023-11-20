FROM golang:latest as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

FROM alpine:latest

RUN apk --no-cache upgrade && apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/certs ./certs
COPY --from=builder /app/utils ./utils
COPY entrypoint.sh ./

RUN chmod +x entrypoint.sh

VOLUME /app/certs

EXPOSE 7000

ENTRYPOINT ["./entrypoint.sh"]
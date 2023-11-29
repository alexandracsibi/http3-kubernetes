FROM golang:latest as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/server .

COPY --from=builder /app/certificate.pem .
COPY --from=builder /app/certificate.key .

EXPOSE 7005

ENTRYPOINT ["./server"]

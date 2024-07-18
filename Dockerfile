FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /go-main ./cmd/yalt/main.go

FROM alpine:latest

WORKDIR /root/

EXPOSE 8080

COPY --from=builder /go-main .

CMD ["./go-main"]

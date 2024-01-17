FROM golang:1.21.1-alpine as builder
WORKDIR /mnt/spacelift-coding-challenge
COPY go.mod .
RUN go mod download
COPY . .
RUN go build -o bin/gateway cmd/gateway/main.go

FROM scratch
COPY --from=builder /mnt/spacelift-coding-challenge/bin/gateway /usr/local/bin/gateway
EXPOSE 3000
ENTRYPOINT ["/usr/local/bin/gateway"]

FROM golang:1.22 AS builder

WORKDIR /app

COPY go.mod go.sum ./ 
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /submissions-bootstrap-node ./cmd/main.go

FROM alpine:latest AS final

WORKDIR /root/

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /submissions-bootstrap-node .

EXPOSE 4001

ENTRYPOINT ["./submissions-bootstrap-node"]

FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /2mon ./cmd/server/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY certs/russian_trusted_root_ca_pem.crt /usr/local/share/ca-certificates/russian_trusted_root_ca.crt
COPY certs/russian_trusted_sub_ca_pem.crt /usr/local/share/ca-certificates/russian_trusted_sub_ca.crt
RUN update-ca-certificates
COPY --from=builder /2mon /2mon
COPY web/ /web/

EXPOSE 8080
CMD ["/2mon"]

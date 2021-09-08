FROM golang:latest as builder
RUN apt-get update && apt-get install -y nocache git ca-certificates && update-ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
#RUN go env -w GOPROXY="https://goproxy.io,direct"
RUN go mod download
COPY . .
RUN go build -o /app/bin/klovercloud-queue .



FROM debian:buster-slim
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
WORKDIR /app
COPY --from=builder /app/bin /app
EXPOSE 8080
# Run the executable
CMD ["./klovercloud-queue"]
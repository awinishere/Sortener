FROM golang:1.26.9-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/sortener ./cmd

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /out/sortener /sortener

ENV PORT=:8080

EXPOSE 8080

USER 65534:65534

ENTRYPOINT ["/sortener"]

FROM golang:1.18-alpine as builder

WORKDIR /app 

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false ./cmd/bgweb-api

FROM scratch

WORKDIR /app

COPY --from=builder /app/bgweb-api /usr/bin/
COPY --from=builder /app/cmd/bgweb-api/data/ /var/lib/bgweb-api/data/

ENTRYPOINT ["bgweb-api", "--datadir", "/var/lib/bgweb-api/data"]

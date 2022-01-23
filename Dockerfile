FROM golang:1.17-alpine as builder

WORKDIR /app 

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" .

FROM scratch

WORKDIR /app

COPY --from=builder /app/bgweb-api /usr/bin/
COPY --from=builder /app/data/ /var/lib/bgweb-api/data/

ENV BGWEB_DATADIR=/var/lib/bgweb-api/data

ENTRYPOINT ["bgweb-api"]

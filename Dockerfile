FROM golang:1.24.2-alpine3.20 AS builder
WORKDIR /go/src/github.com/mrueg/netcupscp-exporter
COPY . .
RUN apk --no-cache add make git && make

FROM alpine:3.21
COPY --from=builder /go/src/github.com/mrueg/netcupscp-exporter/netcupscp-exporter /

CMD ["/netcupscp-exporter"]

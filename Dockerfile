FROM golang:1.24.3-alpine3.21 AS builder
WORKDIR /go/src/github.com/mrueg/netcupscp-exporter
COPY . .
RUN apk --no-cache add make git && make

FROM alpine:3.22
COPY --from=builder /go/src/github.com/mrueg/netcupscp-exporter/netcupscp-exporter /

CMD ["/netcupscp-exporter"]

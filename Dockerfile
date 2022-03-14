FROM golang:1.17.8-alpine3.15 AS builder
WORKDIR /go/src/github.com/mrueg/netcupscp-exporter
COPY . .
RUN apk --no-cache add make git && make

FROM scratch
COPY --from=builder /go/src/github.com/mrueg/netcupscp-exporter/netcupscp-exporter /

CMD ["/netcupscp-exporter"]

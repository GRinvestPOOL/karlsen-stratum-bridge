# Many thanks to original author Brandon Smith (onemorebsmith).
FROM golang:1.19.1 as builder

LABEL org.opencontainers.image.description="Dockerized Nexellia Stratum Bridge"
LABEL org.opencontainers.image.authors="Nexellia Community"
LABEL org.opencontainers.image.source="https://github.com/GRinvestPOOL/nexellia-stratum-bridge"

WORKDIR /go/src/app
ADD go.mod .
ADD go.sum .
RUN go mod download

ADD . .
RUN go build -o /go/bin/app ./cmd/nexelliabridge


FROM gcr.io/distroless/base:nonroot
COPY --from=builder /go/bin/app /
COPY cmd/nexelliabridge/config.yaml /

WORKDIR /
ENTRYPOINT ["/app"]

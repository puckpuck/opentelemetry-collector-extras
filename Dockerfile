FROM --platform=$BUILDPLATFORM golang:1.24-bullseye AS builder
ARG TARGETOS
ARG TARGETARCH

RUN go install go.opentelemetry.io/collector/cmd/builder@v0.139.0

WORKDIR /

COPY metricsasattributesprocessor ./metricsasattributesprocessor
COPY ocb.yaml ./ocb.yaml

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} builder --config=ocb.yaml

FROM --platform=$BUILDPLATFORM alpine:3.16 AS certs
RUN apk --update add ca-certificates

FROM scratch

ARG USER_UID=10001
USER ${USER_UID}

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder --chmod=755 /dist/otelcol-custom /otelcol-custom
EXPOSE 4317 4318
ENTRYPOINT ["/otelcol-custom"]
CMD ["--config", "/etc/otel/config.yaml"]

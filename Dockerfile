FROM golang:1.23.2 AS builder
COPY . /app
WORKDIR /app

ARG VERSION="unknown"
ARG BUILD_DATE=""
ARG BUILD_AGENT="Unknown"
ARG HASH="xxxxxxxx"
ARG FLAGS="-X 'main.Version=${VERSION}' -X 'main.Hash=${HASH}' -X 'main.BuildDate=${BUILD_DATE}' -X 'main.BuildAgent=${BUILD_AGENT}'"
RUN CGO_ENABLED=0 go build -mod=vendor -ldflags="$FLAGS" ngress/cmd/ngress


FROM scratch
# next string prevents: 'x509: certificate signed by unknown authority' error, do not remove!
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/ngress ./ngress
ENTRYPOINT ["./ngress"]
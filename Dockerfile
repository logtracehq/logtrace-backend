FROM golang:1.25 AS build-env

WORKDIR /go/logtrace

LABEL org.opencontainers.image.description="Logtrace helps businesses stay compliant"

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

ARG VERSION=dev
ARG COMMIT=none

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT}" \
    -o /go/bin/logtrace ./cmd/...

FROM gcr.io/distroless/static-debian12

COPY --from=build-env /go/bin/logtrace /logtrace

EXPOSE 8080

ENTRYPOINT ["/logtrace"]
CMD ["http"]

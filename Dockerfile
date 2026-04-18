FROM golang:1.25 AS build-env
WORKDIR /go/logtrace

LABEL org.opencontainers.image.description="Logtrace helps businesses stay compliant"

COPY ./go.mod /go/logtrace
COPY ./go.sum /go/logtrace

RUN go mod download
RUN go mod verify
COPY . .

ARG VERSION=dev
ARG COMMIT=none

RUN CGO_ENABLED=0
RUN go install -ldflags="-X main.Version=${VERSION} -X main.Commit=${COMMIT}" ./cmd/...

FROM busybox:1.37.0-uclibc as busybox

FROM gcr.io/distroless/base

COPY --from=busybox /bin/sh /bin/sh

COPY --from=build-env /go/bin/cmd /
CMD ["/cmd", "http"]

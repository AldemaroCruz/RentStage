# syntax=docker/dockerfile:1.7
FROM golang:1.27rc3-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Some pre-v0.13 source packages generated Firebase's transitive module sums
# only inside Docker. Tidy before tests so the build remains recoverable, while
# the CI backend job separately requires go.mod/go.sum to be committed cleanly.
RUN go mod tidy && go mod verify
RUN test -z "$(gofmt -l .)"
RUN CGO_ENABLED=0 GOOS=linux go test -count=1 ./...
RUN CGO_ENABLED=0 GOOS=linux go build \
    -mod=readonly \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/rentstage-api \
    ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/rentstage-api /app/rentstage-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/rentstage-api"]

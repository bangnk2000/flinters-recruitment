# Build a static binary, then ship it alone in a distroless image.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /aggregator ./cmd/aggregator

FROM gcr.io/distroless/static-debian12
COPY --from=build /aggregator /aggregator
ENTRYPOINT ["/aggregator"]

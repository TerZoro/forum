FROM golang:1.24-alpine AS build

WORKDIR /app

RUN apk add --no-cache build-base sqlite-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o /app/forum ./cmd/web

FROM alpine:3.23 AS production

RUN apk add --no-cache ca-certificates sqlite-libs

RUN addgroup -S nonroot && adduser -S nonroot -G nonroot

WORKDIR /app

COPY --from=build --chown=nonroot:nonroot /app/forum ./forum
COPY --from=build --chown=nonroot:nonroot /app/static ./static
COPY --from=build --chown=nonroot:nonroot /app/templates ./templates
RUN chown -R nonroot:nonroot /app
USER nonroot

EXPOSE 8080

ENTRYPOINT ["./forum"]

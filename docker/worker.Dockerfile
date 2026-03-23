FROM golang:1.24.2 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/worker ./cmd/worker

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=build /out/worker /app/worker
ENTRYPOINT ["/app/worker"]

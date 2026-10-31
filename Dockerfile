# --- build -------------------------------------------------------------------
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY templates ./templates
COPY assets ./assets
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /poet .

# --- runtime -----------------------------------------------------------------
# distroless static: no shell, ca-certs + a nonroot user, tiny.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /poet /poet
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD ["/poet", "-healthcheck"]
ENTRYPOINT ["/poet"]

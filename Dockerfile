FROM golang:1.19.2-bullseye as build

COPY go.mod src/go.mod
COPY go.sum src/go.sum
RUN cd src/ && go mod download

COPY pkg src/pkg/
COPY cmd src/cmd/

# --mount=type=cache,target=/root/.cache/go-build
RUN cd src && go build -tags osusergo,netgo -o /app cmd/*.go; 



FROM gcr.io/distroless/static

USER nonroot:nonroot

COPY --from=build /app /app

EXPOSE 8000

CMD ["/app"]

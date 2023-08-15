FROM paulcager/go-base:latest as build
WORKDIR /app

COPY go.mod ./
COPY cmd ./cmd
COPY sbs ./sbs
COPY beast ./beast
RUN go mod tidy && go mod download && CGO_ENABLED=0 go build -v -o /dump1090_proxy ./cmd/dump1090-proxy

FROM scratch
WORKDIR /app
COPY --from=build /dump1090_proxy ./
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 9799
CMD ["/app/dump1090_proxy", "--remote=pi-zero-flights.paulcager.org:30005", "--remote=pi-zero-flights-2.paulcager.org:30005"]


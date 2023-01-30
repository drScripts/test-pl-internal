FROM registry.mejik.id/sd/base-image as base

WORKDIR /go/synchrodb/services

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o ./app cmd/main.go

FROM scratch

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=base /go/synchrodb/services/app ./

ENTRYPOINT ["./app"]

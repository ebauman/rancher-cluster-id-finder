FROM golang:1.19-buster AS build

WORKDIR /app

COPY . .

RUN go get -d -v ./...

RUN go build -v -o rcidf ./...

FROM gcr.io/distroless/base-debian11 AS run

COPY --from=build /app/rcidf /rcidf

ENTRYPOINT ["/rcidf"]
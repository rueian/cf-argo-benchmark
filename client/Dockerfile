FROM golang:1.13-buster as build
WORKDIR /client
COPY . .
RUN go build -o /main ./main.go

FROM gcr.io/distroless/base-debian10
COPY --from=build /main ./main
CMD ["./main"]
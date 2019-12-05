FROM golang:1.13-buster as build
COPY . /
RUN go build -o /main /main.go

FROM gcr.io/distroless/base-debian10
COPY --from=build /fixtures ./fixtures
COPY --from=build /main ./main
CMD ["./main"]
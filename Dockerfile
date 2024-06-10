FROM golang:1.22 as builder

ARG TARGETARCH

WORKDIR /go/src

COPY . .

RUN go mod download
RUN mkdir /data

RUN GOOS=linux GOARCH=${TARGETARCH} go build -ldflags "-s -w" -o entropy .

# Path: Dockerfile
FROM scratch

WORKDIR /bin

COPY --from=builder /go/src/entropy /bin
COPY --from=builder /data /data

ENTRYPOINT [ "entropy" ] 





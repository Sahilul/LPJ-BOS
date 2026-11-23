FROM golang:1.26-alpine AS build
WORKDIR /src
COPY src/go.mod src/go.sum ./
RUN go mod download
COPY src/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/lpjbos ./cmd/lpjbos

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/lpjbos /usr/local/bin/lpjbos
ENV DATA_DIR=/var/www/data PORT=80 TZ=Asia/Jakarta
VOLUME /var/www/data
EXPOSE 80
CMD ["lpjbos"]

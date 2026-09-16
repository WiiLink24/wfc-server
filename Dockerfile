FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /build/wwfc .

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=builder /build/wwfc /app/wwfc
COPY game_list.tsv motd.txt /app/

# config.xml and all certificates/keys are deliberately NOT baked into the
# image, but mounted at runtime instead (see docker-compose.yml). This keeps
# the image generic, secret-free and reusable for any deployment (Pi, PC,
# etc.) without a rebuild.

EXPOSE 80 443 28910 29900 29901 29920 27900/udp 27901/udp

CMD ["/app/wwfc"]

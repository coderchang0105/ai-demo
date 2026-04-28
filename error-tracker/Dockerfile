# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -o server ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && apk add --no-cache gcc musl-dev

WORKDIR /app

COPY --from=builder /app/server .
COPY web/ ./web/
COPY example/ ./example/

EXPOSE 8080

CMD ["./server"]

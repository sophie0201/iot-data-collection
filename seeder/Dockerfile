FROM golang:1.26-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git

# 複製 go.mod 和 go.sum（如果有的話）
COPY go.mod go.sum ./
RUN go mod download

# 複製 seeder 程式碼
COPY seeder/ ./seeder/

# 編譯 seeder
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o seeder ./seeder

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

ENV TZ=Asia/Taipei

WORKDIR /root/

COPY --from=builder /build/seeder .

CMD ["./seeder"]

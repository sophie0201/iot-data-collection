FROM golang:1.26-alpine AS builder

WORKDIR /build  

RUN apk add --no-cache git

COPY go.mod go.sum ./

RUN go mod download

COPY app/ ./app/

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./app

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

ENV TZ=Asia/Taipei

WORKDIR /root/

COPY --from=builder /build/main .

EXPOSE 8080

CMD ["./main"]

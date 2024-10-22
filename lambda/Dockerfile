FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY . .

ARG GOPROXY
ARG GONOPROXY
ARG GOINSECURE

ENV GOPROXY="${GOPROXY}"
ENV GONOPROXY="${GONOPROXY}"
ENV GOINSECURE="${GOINSECURE}"

RUN apk add git \
    && git config --global http.sslVerify false

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main .

FROM gcr.io/distroless/static-debian12

COPY --from=builder /app/main /app/main

EXPOSE 8081

CMD ["/app/main"]

FROM gitea.starryskymeow.cn/cn/debian:trixie AS builder

WORKDIR /app

RUN apt update && apt install -y golang git && apt clean

COPY go.sum go.mod . 

RUN go mod download

# copy all for version
COPY . . 

RUN go build -o /app/server ./cmd/server

FROM gitea.starryskymeow.cn/cn/debian:trixie AS runner

WORKDIR /app

RUN apt update && apt install -y ca-certificates && apt clean && rm -rf /var/lib/apt/lists

COPY --from=builder /app/server /app/server

COPY --from=builder /app/web /app/web

EXPOSE 8080

CMD ["/app/server"]

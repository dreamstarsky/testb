FROM reg.xd.xkm.be/xdu/debian:trixie

WORKDIR /app

RUN apt update && apt install -y golang && apt clean

COPY go.sum go.mod . 

RUN go mod download

COPY . . 

RUN go build ./cmd/server

EXPOSE 8080

CMD /app/server

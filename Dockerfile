FROM golang:latest

# go get
RUN go get github.com/bwmarrin/discordgo

# source get
RUN mkdir /marumesi
COPY ./weather.go /marumesi/
COPY ./main.go /marumesi/
COPY ./devcmd.go /marumesi/
COPY ./setting.json /marumesi/

# build
WORKDIR /marumesi
RUN go build
ENTRYPOINT /marumesi/marumesi
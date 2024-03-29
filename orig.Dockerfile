FROM golang:alpine
RUN apk add git
RUN mkdir /app
COPY . /app
WORKDIR /app
RUN go get github.com/gorilla/sessions
RUN go get github.com/nfnt/resize
RUN go get golang.org/x/oauth2
RUN go get golang.org/x/oauth2/google
RUN go get github.com/mholt/certmagic
RUN go get github.com/postmannen/authsession
RUN go get github.com/boltdb/bolt
RUN go build -o main .
# CMD ["/app/main", "-proto=http://", "-host=eyer.io", "-port=80", "-hostListen=0.0.0.0"]
CMD ["/app/main", "-proto=https", "-host=eyer.io", "-port=443"]
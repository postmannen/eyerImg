FROM golang:alpine
RUN apk add git
RUN mkdir /app
COPY . /app
WORKDIR /app
RUN go get github.com/gorilla/sessions
RUN go get github.com/nfnt/resize
RUN go get golang.org/x/oauth2
RUN go get golang.org/x/oauth2/google
RUN go build -o main .
CMD ["/app/main"]
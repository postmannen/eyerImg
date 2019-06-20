# eyerImg

In progress.
Web service with Authentication against Google, and image upload to local disk on server.

## To use

The repository includes a Dockerfile

Replaces the CMD values to fit your setup.

```Dockerfile
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
CMD ["/app/main", "-proto=http://", "-host=eyer.io", "-port=:80", "-hostListen=0.0.0.0"]
```

## Environment variables

The programs expects the following  environment variables set.

```
cookiestorekey=some-cookie-store-key-here
googlekey=some-google-key-here
googlesecret=some-google-secret-here
```

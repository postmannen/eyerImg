# eyerImg

In progress.
Web service with Authentication against Google, and image upload to local disk on server.

## To use

Start locally, just http.
`go run *go -host=localhost -hostListen=localhost -port="8080" -proto="http"`

Start externally with https, will also redirect http call to https.
`go run *go -host=eyer.io -port=443 -proto=https`
We don't have to specify the -hostListen when running https.

## To use with Docker

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
CMD ["/app/main", "-proto=http", "-host=eyer.io", "-port=:80", "-hostListen=0.0.0.0"]
```

To build image
`docker build -t eyerimg .`

## Environment variables

The programs expects the following  environment variables set for authentication against Google Cloud.

```
cookiestorekey=some-cookie-store-key-here
googlekey=some-google-key-here
googlesecret=some-google-secret-here
```

The variables can be stored in a file called for example `exports`, and can then be implemented when starting the container like this.

`docker run --env-file exports -p 80:80 eyerimg`

## Google Cloud specific things

Create an Oauth app under app credentials i Google Cloud Console.
Set the callback URL to

`http://<your domain name>:<some port>/callback`

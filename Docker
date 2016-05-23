FROM golang:1.6.2-wheezy
MAINTAINER Vladislav Spirenkov <moiplov@gmail.com>

# Install system dependencies
RUN apt-get update -qq && \
    apt-get install -qq -y pkg-config build-essential

RUN mkdir -p /app
WORKDIR /app
COPY . /app/
ENV GOPATH /go/
RUN go get -d -v
RUN go build -o letsconsul

CMD ["./letsconsul"]

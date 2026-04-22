FROM golang:1.26-alpine

RUN apk add --no-cache fish curl
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin

WORKDIR /mnt
COPY . .

CMD ["/usr/bin/fish"]
CMD ["ls", "-la", "/mnt"]

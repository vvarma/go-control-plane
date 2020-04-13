FROM golang

WORKDIR /go-control-plane

# Fetch and preserve module dependencies
ENV GOPROXY=https://proxy.golang.org
COPY go.mod ./
RUN go mod download

COPY . ./
RUN go install ./cmd/server

CMD /go/bin/server
# Using Alpine as builder: ~ 14 MB image
FROM golang:alpine3.15 as builder

RUN apk --update add --no-cache \
        gcc \
	    libc-dev \
	    libpcap-dev

# Using Debian as builder: ~ 15 MB image
# FROM golang:1.18-buster as builder
# RUN apt-get update  && apt-get install -y libpcap-dev

LABEL author="fduran.com"

ARG APP=scanblocker

ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux \ 
    GOARCH=amd64

# Create and change to the app directory.
WORKDIR /app

# Get dependencies go.mod (go.sum)
COPY go.mod go.sum ./
RUN go mod download

# Copy local code to the container image.
COPY ./cmd/${APP}/* ./

# Build the binary.
RUN go build --ldflags '-linkmode external -extldflags "-static"' -o ${APP} .

# distroless.
FROM scratch

# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/${APP} /app/${APP}

# Run the app on container startup.
# needs the brackets since distroless (no shell interpretation).
CMD ["/app/scanblocker"]
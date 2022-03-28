# Go PATH
export PATH := /usr/local/go/bin:/usr/bin/gcc:$(PATH)

# to pass current user env var:
# sudo -E make run

# Docker registry
REPO := fduran

# binary name (can add arch)
BIN := scanblocker

# target
TARGET := scanblocker

SRC_DIRS := cmd# pkg

#ALL_PLATFORMS := linux/amd64
#OS := linux
#ARCH := amd64


# make without args runs first target
# -ldflags "-X main.GitCommit=$GIT_COMMIT" optionally
# --ldflags '-linkmode external -extldflags "-static"'

# https://docs.docker.com/develop/develop-images/build_enhancements/
EXPORT := DOCKER_BUILDKIT=1

build:
	go build -o $(BIN) $(SRC_DIRS)/$(TARGET)/*.go

run: build
	./$(BIN)

dep:
	go mod download

clean:
	go clean

test:
	go test ./... -v

fmt:
	go fmt ./...

vet:
	go vet

lint:
	gofmt -w .
	golangci-lint run --enable-all

image:
	docker build -t $(REPO)/$(TARGET) .
	docker tag $(REPO)/$(TARGET) $(TARGET)

# see also https://goreleaser.com/
release: image
	git tag -a $(VERSION) -m "Release" || true
	git push origin $(VERSION)

# dir names conflict
.PHONY: build test


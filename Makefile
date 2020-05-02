SERVICE_NAME=deployments
GO=go
GOLIST=$(GO) list ./...
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOTEST=$(GO) test

all: test build

build: 
		CGO_ENABLED=1 $(GOBUILD) -o $(SERVICE_NAME)

test: 
		$(GOLIST) | grep -v /vendor | xargs -n1 -I {} -P 4 $(GOTEST) -v {}

clean: 
		$(GOCLEAN)
		rm -f $(SERVICE_NAME)

run: build
		./$(SERVICE_NAME)

run-automigrate: build
		./$(SERVICE_NAME) server --automigrate


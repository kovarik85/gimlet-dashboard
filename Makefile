	GO_VERSION=1.16
GOFILES = $(shell find . -type f -name '*.go' -not -path "./.git/*")
LDFLAGS = '-s -w -extldflags "-static" -X github.com/gimlet-io/gimlet-dashboard/version.Version='${VERSION}

DOCKER_RUN?=
_with-docker:
	$(eval DOCKER_RUN=docker run --rm -v $(shell pwd):/go/src/github.com/gimlet-io/gimlet-dashboard -w /go/src/github.com/gimlet-io/gimlet-dashboard golang:$(GO_VERSION))

.PHONY: all format-backend test-backend test-frontend build-frontend build-backend build-agent dist

format-backend:
	@gofmt -w ${GOFILES}

test-backend:
	$(DOCKER_RUN) go test -race -timeout 60s $(shell go list ./... )

build-backend:
	$(DOCKER_RUN) CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -o build/gimlet-dashboard github.com/gimlet-io/gimlet-dashboard/cmd/dashboard

build-agent:
	$(DOCKER_RUN) CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -o build/gimlet-agent github.com/gimlet-io/gimlet-dashboard/cmd/agent

dist:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -ldflags $(LDFLAGS) -a -installsuffix cgo -o bin/gimlet-dashboard-linux-x86_64 github.com/gimlet-io/gimlet-dashboard/cmd/dashboard
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags $(LDFLAGS) -a -installsuffix cgo -o bin/gimlet-agent-linux-x86_64 github.com/gimlet-io/gimlet-dashboard/cmd/agent

build-frontend:
	(cd web/; npm install; npm run build)

test-frontend:
	(cd web/; npm install; npm run test)

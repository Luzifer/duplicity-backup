default:

install:
	go install -a -ldflags "-X main.version=$(shell git describe --tags || git rev-parse --short HEAD || echo dev)"

lint:
	golangci-lint run ./...

test:
	go test -v ./...

publish:
	curl -sSLo golang.sh https://raw.githubusercontent.com/Luzifer/github-publish/master/golang.sh
	bash golang.sh

# -- Vulnerability scanning --

trivy:
	trivy fs . \
		--dependency-tree \
		--exit-code 1 \
		--format table \
		--ignore-unfixed \
		--quiet \
		--scanners config,license,secret,vuln \
		--severity HIGH,CRITICAL \
		--skip-dirs docs

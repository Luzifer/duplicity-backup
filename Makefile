default:

ci: publish

publish:
	curl -sSLo golang.sh https://raw.githubusercontent.com/Luzifer/github-publish/master/golang.sh
	bash golang.sh

test:
		go test -v

install:
		go install -a -ldflags "-X main.version=$(shell git describe --tags || git rev-parse --short HEAD || echo dev)"

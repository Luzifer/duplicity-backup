bindata:
		go-bindata help.txt

ci: publish

publish:
	curl -sSLo golang.sh https://raw.githubusercontent.com/Luzifer/github-publish/master/golang.sh
	bash golang.sh

setup-testenv:
		go get github.com/onsi/ginkgo/ginkgo
		go get github.com/onsi/gomega
		go get github.com/alecthomas/gometalinter
		gometalinter --install --update

test:
		go test -v

install:
		go install -a -ldflags "-X main.version=$(shell git describe --tags || git rev-parse --short HEAD || echo dev)"

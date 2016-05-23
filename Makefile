bindata:
		go-bindata help.txt

setup-testenv:
		go get github.com/onsi/ginkgo/ginkgo
		go get github.com/onsi/gomega
		go get github.com/alecthomas/gometalinter
		gometalinter --install --update

test:
		ginkgo
		gometalinter \
				--cyclo-over=15 \
				--deadline=20s \
				--exclude=bindata.go \
				--exclude=configfile_test.go \
				-D errcheck

install:
		go install -a -ldflags "-X main.version=$(shell git describe --tags || git rev-parse --short HEAD || echo dev)"

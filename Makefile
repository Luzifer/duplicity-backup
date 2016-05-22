bindata:
		go-bindata help.txt

setup-testenv:
		go get github.com/onsi/ginkgo/ginkgo
		go get github.com/onsi/gomega

test:
		$(GOPATH)/bin/ginkgo

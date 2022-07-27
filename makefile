
install-go-modules:
	go mod tidy

test/e2e: install-go-modules
	sh ./test/e2e/e2e.sh

test/template: install-go-modules
	go test -v ./test/template/...
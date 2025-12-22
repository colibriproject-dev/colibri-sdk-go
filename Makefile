fmt:
	go fmt ./...

mock:
	find . -type f -name "*_mock.go" -exec rm -f {} \;
	go generate -v ./...

test: mock
	mkdir -p coverage
	go test -cover ./... -args -test.gocoverdir="${PWD}/coverage/"

cover:
	go tool covdata textfmt -i=./coverage -o coverage.txt
	go tool cover -html coverage.txt

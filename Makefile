fmt:
	go fmt ./...

mock:
	find . -type f -name "*_mock.go" -exec rm -f {} \;
	go generate -v ./...

test: mock
	mkdir -p coverage
	go test -timeout 10m -cover ./... -args -test.gocoverdir="${PWD}/coverage/"

metrics-doc:
	go run ./internal/tools/metricsdoc

PROMTOOL=docker run --rm -v "${PWD}/observability/alerts:/rules:ro" -w /rules --entrypoint promtool prom/prometheus:v3.14.0

observability: observability-check observability-rules

observability-check:
	cd observability && go test ./...

observability-rules:
	${PROMTOOL} check rules rules.yaml
	${PROMTOOL} test rules rules_test.yaml

cover:
	go tool covdata textfmt -i=./coverage -o coverage.txt
	go tool cover -html coverage.txt

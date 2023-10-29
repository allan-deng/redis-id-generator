.PHONY: build
BIN_FILE=idgensvr

build:
	go version
	go build -o "${BIN_FILE}" ./cmd/main.go

test: 
	go test -cover -coverprofile=coverage.out -v -timeout 60s ./pkg/idgen
	
bench:
	go test -benchmem -run=^$$ -bench=^Benchmark  -benchtime=5s -cpu=1,2,4,8,16 github.com/allan-deng/redis-id-generator/pkg/idgen

package:
	sh ./script/package.sh
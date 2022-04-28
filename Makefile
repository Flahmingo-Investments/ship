.PHONY: schema
schema:
	rm -rf ./schema/ship.pb.go
	protoc \
		-I ./schema/ \
		--go_out="paths=source_relative:./schema" \
		ship.proto
	
.PHONY: lint
lint:
	golangci-lint run

test:
	go test -race -coverprofile coverage.out ./...
	go tool cover -func coverage.out

generate/mock:
	go generate ./...

example: protoc-gen-go bin/protoc-gen-ship
	rm -rf ./example/generated && mkdir -p ./example/generated
	set -e; for subdir in `find ./example -mindepth 0 -maxdepth 1 -type d`; do \
	 	files=`find $$subdir -maxdepth 1 -name "*.proto"`; \
	 	[ ! -z "$$files" ] && \
		protoc \
			-I ./ \
			-I ./example/ \
			--plugin=protoc-gen-ship=./bin/protoc-gen-ship \
	 		--go_out="./example/generated" \
			--ship_out="./example/generated" \
			$$files; \
	done
	

.PHONY: protoc-gen-go
protoc-gen-go:
	which protoc-gen-go || (go install github.com/golang/protobuf/protoc-gen-go)

.PHONY: bin/*
bin/protoc-gen-ship:
	go build -o ./bin/protoc-gen-ship ./protoc-gen-ship

.PHONY: clean
clean:
	rm -rf bin
	rm -rf example/generated

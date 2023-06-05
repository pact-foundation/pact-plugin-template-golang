TEST?=./...
.DEFAULT_GOAL := ci
FFI_VERSION=0.3.15
# Update this version. It will be sourced
VERSION?=0.0.1
# Update to your project name
PROJECT=myplugin

ci:: deps clean bin test

bin: write_config
	go build -o build/$(PROJECT)

clean:
	rm -rf build dist

deps:
	@echo "--- 🐿  Fetching build dependencies "
	cd /tmp; \
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 ;\
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2 ;\
	cd -

test: deps
	go test $(TEST)

proto:
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		io_pact_plugin/pact_plugin.proto

install_local: bin write_config
	@echo "Creating a local phony plugin install so we can test locally"
	mkdir -p ~/.pact/plugins/$(PROJECT)-$(VERSION)
	cp ./build/$(PROJECT) ~/.pact/plugins/$(PROJECT)-$(VERSION)/
	cp pact-plugin.json ~/.pact/plugins/$(PROJECT)-$(VERSION)/

write_config:
	@cp pact-plugin.json pact-plugin.json.new
	@cat pact-plugin.json | jq '.version = "'$(subst v,,$(VERSION))'" | .name = "'$(PROJECT)'" | .entryPoint = "'$(PROJECT)'"' | tee pact-plugin.json.new
	@mv pact-plugin.json.new pact-plugin.json 

ffi:
	FFI_VERSION=$(FFI_VERSION) ./scripts/download-libs.sh

.PHONY: bin test clean write_config


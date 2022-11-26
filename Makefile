TEST?=./...
NAME?=matt
.DEFAULT_GOAL := ci
PACT_CLI="docker run --rm -v ${PWD}:${PWD} -e PACT_BROKER_BASE_URL=$(DOCKER_HOST_HTTP) -e PACT_BROKER_USERNAME -e PACT_BROKER_PASSWORD pactfoundation/pact-cli"

ci:: deps clean bin test

bin:
	go build -o build/$(NAME)

clean:
	rm -rf build

deps:
	@echo "--- 🐿  Fetching build dependencies "
	cd /tmp; \
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 ;\
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2 ;\
	cd -

release:
	echo "--- 🚀 Releasing it"
	"$(CURDIR)/scripts/release.sh"

test: deps install
	go test $(TEST)

testrace:
	go test -race $(TEST) $(TESTARGS)

proto:
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		io_pact_plugin/pact_plugin.proto

install_local: bin 
	@echo "Creating a local phony plugin install so we can test locally"
	mkdir -p ~/.pact/plugins/$(NAME)-0.0.4
	cp ./build/$(NAME) ~/.pact/plugins/$(NAME)-0.0.4/
	cp pact-plugin.json ~/.pact/plugins/$(NAME)-0.0.4/

.PHONY: install bin test clean release

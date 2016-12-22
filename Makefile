GO_LDFLAGS=-ldflags " -w"

TAG=v1
PREFIX=barnettzqg/tcm
Name=tcm

build: ## build the go packages
	@echo "🐳 $@"
	@go build -a -installsuffix cgo ${GO_LDFLAGS} .

image:
	@echo "🐳 $@"
	@docker build -t $(PREFIX):$(TAG) .
    # @docker push $(PREFIX):$(TAG)
clean:
	@rm -f ${Name}
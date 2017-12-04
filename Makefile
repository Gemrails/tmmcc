TAG=v1
PREFIX=barnettzqg/tcm
Name=tcm

build: ## build the go packages
	@echo "🐳 $@"
	@go build -o tcm ./cmd
run:build
	@sudo ./tcm -i lo0
image:
	@echo "🐳 $@"
	@docker build -t $(PREFIX):$(TAG) .
    # @docker push $(PREFIX):$(TAG)
clean:
	@rm -f ${Name}
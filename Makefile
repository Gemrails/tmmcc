TAG=v1
PREFIX=hub/tcm
Name=tcm

build: ## build the go packages
	@echo "🐳 $@"
	@go build -o tcm ./cmd
run:build
	sudo ./tcm -i lo0 -protocol mysql  -expr="tcp port 3306"
image:
	@echo "🐳 $@"
	@docker build -t $(PREFIX):$(TAG) .
    # @docker push $(PREFIX):$(TAG)
clean:
	@rm -f ${Name}
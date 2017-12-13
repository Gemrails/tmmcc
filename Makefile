TAG=v1
PREFIX=rainbond/tcm
Name=tcm

build: ## build the go packages
	@echo "🐳 $@"
	@go build -o tcm ./cmd
run:build
	sudo ./tcm -i lo0 -protocol mysql  -expr="tcp port 3306"
dockerbuild:
	@echo "🐳 $@"
	@docker build -t tcmbuild ./build
	@docker run -v `pwd`:/go/src/tcm -w /go/src/tcm --rm -it tcmbuild go build -o tcm ./cmd
image:dockerbuild
	@echo "🐳 $@"
	@docker build -t $(PREFIX):$(TAG) .
    # @docker push $(PREFIX):$(TAG)
	clean
clean:
	@rm -f ${Name}
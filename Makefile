TAG=v1
PREFIX=rainbond/tcm
Name=tcm

localbuild: ## build the go packages
	@echo "🐳 $@"
	@go build -o tcm ./cmd
run:localbuild
	sudo ./tcm -i lo0 -protocol http  -expr="tcp port 5000"

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
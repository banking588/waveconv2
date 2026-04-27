# Main Paths
ALL_CMD_PATHS := $(dir $(shell shopt -s globstar; ls cmd/**/main.go))

# make cmd
MAKE := make --no-print-directory

.PHONY: all clean cleanall test cmdtest generate $(ALL_CMD_PATHS)

all: tidy $(ALL_CMD_PATHS)

$(ALL_CMD_PATHS):
	@echo "# make $@..."
	@cd $@ && $(MAKE)

clean: $(addsuffix -clean,$(ALL_CMD_PATHS)) 
%-clean:
	@echo "# clean $*..."
	@cd $* && $(MAKE) clean

cleanall: clean
	@echo "# cleanall ..."
	@rm -rf dist tmp vendor

tidy:
	go mod tidy

initenv:
	go env -w CGO_ENABLED=0
	go env -w GOSUMDB=off
	go env -w GOPRIVATE=innotron.com,cxmt.com,*.innotron.com,*.cxmt.com
	go env -w GOPROXY=http://172.16.13.24/repository/go/,https://goproxy.cn,https://goproxy.io,direct
	go install github.com/gogo/protobuf/protoc-gen-gofast@latest

test:
	@echo "# start unit test ..."
	@go test -v -count=1 ./...
	@$(MAKE) cmdtest

cmdtest: $(addsuffix -test,$(ALL_CMD_PATHS)) 	
%-test:
	@echo "# start test $*..."
	@cd $* && $(MAKE) test

# default setting
PROG_NAME ?= $(notdir $(shell pwd))
TARGET_PLATFORM ?= windows-64 linux-64
GOCOMPILER ?= go
DIST_PATH ?= $(WORKSPACE_ROOT)/dist
CLEAN_PATH := $(WORKSPACE_ROOT)/tmp
UPX_ENABLE ?= false
ARCHIVE_ENABLE ?= true
CGO_ENABLED ?= 0
export CGO_ENABLED

# windows path convert
ifdef OS
PWD := $(shell cygpath -m `pwd`)
WORKSPACE_ROOT := $(shell cygpath -m $(WORKSPACE_ROOT))
DIST_PATH := $(shell cygpath -m $(DIST_PATH))
CLEAN_PATH := $(shell cygpath -m $(CLEAN_PATH))
endif

VERSION := $(shell (cat ver.txt || git describe || echo v0.0.0) 2> /dev/null)
BUILD_TIME := $(shell date '+%Y%m%d%H%M%S')
GITHASH := $(shell git rev-parse --short HEAD)

ifneq ($(DEBUG),true)
LDFLAGS := -trimpath -ldflags '-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.ProgName=$(PROG_NAME) -X main.GitHash=$(GITHASH)'
TARGET_DIR_PATH := $(DIST_PATH)/$(PROG_NAME)/$(VERSION)-$(GITHASH)-$(BUILD_TIME)
else
LDFLAGS := -gcflags='all=-N -l' -ldflags='-compressdwarf=false -X main.Version=debug'
TARGET_DIR_PATH := $(DIST_PATH)/$(PROG_NAME)/debug
endif

BUILD_CMD = GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOENV) $(GOCOMPILER) build $(BUILDFLAG) $(LDFLAGS) -o $(TARGET_DIR_PATH)/$@$(BINARY_EXTENTION)

.PHONY: all
all: $(TARGET_PLATFORM)

# append PROG_NAME and TARGET_PLATFORM
$(TARGET_PLATFORM): % : $(PROG_NAME)-%

# build target
$(PROG_NAME)-%: history prebuild copy_res
	@echo "build $@$(BINARY_EXTENTION)..."
	$(BUILD_CMD)
	@echo "postbuild..."
	$(POST_CMD)
ifneq ($(DEBUG),true)
ifeq ($(UPX_ENABLE),true)
	@echo "compress $@$(BINARY_EXTENTION)..."
	@-upx -9 $(TARGET_DIR_PATH)/$@$(BINARY_EXTENTION) > /dev/null
endif
ifeq ($(ARCHIVE_ENABLE),true)
	@echo "archive $(PROG_NAME)-$@.tar.gz..."
	@cd $(TARGET_DIR_PATH) && tar zcf $@.tar.gz $@$(BINARY_EXTENTION) $(notdir $(RESOURCES)) history.txt
endif
	@rm -rf $(DIST_PATH)/$(PROG_NAME)/latest
	@ln -s $(TARGET_DIR_PATH) $(DIST_PATH)/$(PROG_NAME)/latest
endif
	

# build settings for windows64
windows-64: GOOS = windows
windows-64: GOARCH = amd64
windows-64: BINARY_EXTENTION = .exe

# build settings for linux-32
linux-32: GOOS = linux
linux-32: GOARCH = 386

# build settings for linux-64
linux-64: GOOS = linux
linux-64: GOARCH = amd64

# build settings for linux-armv7
linux-armv7: GOOS = linux
linux-armv7: GOARCH = arm
linux-armv7: GOENV = GOARM=7

# build settings for linux-armv8
linux-armv8: GOOS = linux
linux-armv8: GOARCH = arm64

.PHONY:prebuild
prebuild:
	@echo "prebuild..."
	$(PRE_CMD)

.PHONY:copy_res
copy_res:	
	@echo "copy resources..."
	@-mkdir -p $(TARGET_DIR_PATH)
	@if [ ! -z "$(RESOURCES)" ]; then cp -rf $(RESOURCES) $(TARGET_DIR_PATH)/. ; fi

.PHONY:history
history:
	@echo "generate commit history..."
	@-mkdir -p $(TARGET_DIR_PATH)
	git log > $(TARGET_DIR_PATH)/history.txt

.PHONY: clean
clean:
	rm -rf $(DIST_PATH)/$(PROG_NAME) $(CLEAN_PATH)

.PHONY: test
test:

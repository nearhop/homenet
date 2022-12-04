GOMINVERSION = 1.17
NEBULA_CMD_PATH = "./cmd/nebula"
GO111MODULE = on
export GO111MODULE
CGO_ENABLED = 0
export CGO_ENABLED

# Set up OS specific bits
ifeq ($(OS),Windows_NT)
	#TODO: we should be able to ditch awk as well
	GOVERSION := $(shell go version | awk "{print substr($$3, 3)}")
	GOISMIN := $(shell IF "$(GOVERSION)" GEQ "$(GOMINVERSION)" ECHO 1)
	NEBULA_CMD_SUFFIX = .exe
	NULL_FILE = nul
else
	GOVERSION := $(shell go version | awk '{print substr($$3, 3)}')
	GOISMIN := $(shell expr "$(GOVERSION)" ">=" "$(GOMINVERSION)")
	NEBULA_CMD_SUFFIX =
	NULL_FILE = /dev/null
endif

BUILD_NUMBER = $(shell cat buildno.txt)
# Only defined the build number if we haven't already
ifndef BUILD_NUMBER
	BUILD_NUMBER = $(shell date +%Y.%m%d.%H%M%S.0)
endif

LDFLAGS = -X main.Build=$(BUILD_NUMBER)
CFLAGS = -Wno-error=unused-command-line-argument

ALL_LINUX = linux-amd64 \
	linux-amd64-server \
	linux-386 \
	linux-ppc64le \
	linux-arm-5 \
	linux-arm-6 \
	linux-arm-7 \
	linux-arm64 \
	linux-mips \
	linux-mipsle \
	linux-mips64 \
	linux-mips64le \
	linux-mips-softfloat \
	linux-riscv64 \

gl_b1300 = \
	linux-arm-7-gl_b1300 \

asus_ax88u = \
	linux-arm-7-asus_ax88u \

ALL = $(ALL_LINUX) \
	windows-amd64 \
	darwin-amd64 \
	freebsd-amd64 \
	$(gl_b1300) \
	$(asus_ax88u) \

e2e:
	$(TEST_ENV) go test -tags=e2e_testing -count=1 $(TEST_FLAGS) ./e2e

e2ev: TEST_FLAGS = -v
e2ev: e2e

e2evv: TEST_ENV += TEST_LOGS=1
e2evv: e2ev

e2evvv: TEST_ENV += TEST_LOGS=2
e2evvv: e2ev

e2evvvv: TEST_ENV += TEST_LOGS=3
e2evvvv: e2ev

all: $(ALL:%=build/%/nearhopd) $(ALL:%=build/%/nebula-cert)

release: $(ALL:%=build/nebula-%.tar.gz)

release-linux: $(ALL_LINUX:%=build/nebula-%.tar.gz)

release-freebsd: build/nebula-freebsd-amd64.tar.gz

BUILD_ARGS = -trimpath

bin-windows: build/windows-amd64/nebula.exe build/windows-amd64/nebula-cert.exe
	mv $? .

bin-darwin: build/darwin-amd64/nebula build/darwin-amd64/nebula-cert
	mv $? .

bin-freebsd: build/freebsd-amd64/nebula build/freebsd-amd64/nebula-cert
	mv $? .

bin-gl_b1300: build/linux-arm-7-gl_b1300/nearhopd

bin-asus_ax88u: build/linux-arm-7-asus_ax88u/nearhopd

cli-gl_b1300:
	GOOS=linux \
		GOARCH=arm GOARM=7 \
		go build -tags "messages openwrt router" -ldflags "$(LDFLAGS)" -o build/linux-arm-7-gl_b1300/nebula-cli ./cmd/nebula-cli
	cp build/linux-arm-7-gl_b1300/nebula-cli rootfs/linux-arm-7-gl_b1300/usr/sbin/nearhopd-cli

cli-linux-amd64:
	go build $(BUILD_ARGS) -ldflags "$(LDFLAGS)" -o build/linux-amd64/nebula-cli ./cmd/nebula-cli

bin:
	go build $(BUILD_ARGS) -ldflags "$(LDFLAGS)" -o ./nebula${NEBULA_CMD_SUFFIX} ${NEBULA_CMD_PATH}
	go build $(BUILD_ARGS) -ldflags "$(LDFLAGS)" -o ./nebula-cert${NEBULA_CMD_SUFFIX} ./cmd/nebula-cert

install:
	go install $(BUILD_ARGS) -ldflags "$(LDFLAGS)" ${NEBULA_CMD_PATH}
	go install $(BUILD_ARGS) -ldflags "$(LDFLAGS)" ./cmd/nebula-cert

build/linux-arm-7-gl_b1300%: BUILD_ARGS += -tags "messages openwrt router"
build/linux-arm-7-asus_ax88u%: BUILD_ARGS += -tags "messages asus router"
build/linux-arm-%: GOENV += GOARM=$(word 3, $(subst -, ,$*))
build/linux-mips-%: GOENV += GOMIPS=$(word 3, $(subst -, ,$*))
build/windows-amd64%: GOENV += CC="x86_64-w64-mingw32-gcc" CGO_ENABLED=1
build/windows-amd64%: LDFLAGS += -H=windowsgui
build/linux-amd64-%: BUILD_ARGS += -tags $(word 3, $(subst -, ,$*))
build/linux-amd64%: GOENV += CGO_ENABLED=1
build/darwin-amd64%: GOENV += CGO_ENABLED=1 CC=x86_64-apple-darwin20.4-cc
#build/darwin-amd64%: GOENV += CGO_ENABLED=1

build/windows-amd64%: BUILD_ARGS += -tags screen
build/linux-amd64%: BUILD_ARGS += -tags screen
build/darwin-amd64%: BUILD_ARGS += -tags screen

# Build an extra small binary for mips-softfloat
build/linux-mips-softfloat/%: LDFLAGS += -s -w

build/%/nearhopd: .FORCE
	rm -rf rootfs/$*
	rm -rf bin/$*
	mkdir -p rootfs/$*/usr/sbin
	GOOS=$(firstword $(subst -, , $*)) \
		GOARCH=$(word 2, $(subst -, ,$*)) $(GOENV) \
		go build $(BUILD_ARGS) -o $@ -ldflags "$(LDFLAGS)" ${NEBULA_CMD_PATH}
	cp $@ rootfs/$*/usr/sbin/nearhopd
	cp -rL ../nearhop_platform/$*/* rootfs/$* 2>/dev/null || :
	mkdir -p bin/$*
	cd rootfs/$*/; tar -czf ../../bin/$*/$*.tar.gz .

build/%/nebula-cert: .FORCE
	GOOS=$(firstword $(subst -, , $*)) \
		GOARCH=$(word 2, $(subst -, ,$*)) $(GOENV) \
		go build $(BUILD_ARGS) -o $@ -ldflags "$(LDFLAGS)" ./cmd/nebula-cert

build/%/nebula.exe: build/%/nebula
	mv $< $@

build/%/nebula-cert.exe: build/%/nebula-cert
	mv $< $@

build/nebula-%.tar.gz: build/%/nebula build/%/nebula-cert
	tar -zcv -C build/$* -f $@ nebula nebula-cert

build/nebula-%.zip: build/%/nebula.exe build/%/nebula-cert.exe
	cd build/$* && zip ../nebula-$*.zip nebula.exe nebula-cert.exe

vet:
	go vet -v ./...

test:
	go test -v ./...

test-cov-html:
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out

bench:
	go test -bench=.

bench-cpu:
	go test -bench=. -benchtime=5s -cpuprofile=cpu.pprof
	go tool pprof go-audit.test cpu.pprof

bench-cpu-long:
	go test -bench=. -benchtime=60s -cpuprofile=cpu.pprof
	go tool pprof go-audit.test cpu.pprof

proto: nebula.pb.go cert/cert.pb.go

nebula.pb.go: nebula.proto .FORCE
	go build github.com/gogo/protobuf/protoc-gen-gogofaster
	PATH="$(CURDIR):$(PATH)" protoc --gogofaster_out=paths=source_relative:. $<
	rm protoc-gen-gogofaster

cert/cert.pb.go: cert/cert.proto .FORCE
	$(MAKE) -C cert cert.pb.go

service:
	@echo > $(NULL_FILE)
	$(eval NEBULA_CMD_PATH := "./cmd/nebula-service")
ifeq ($(words $(MAKECMDGOALS)),1)
	@$(MAKE) service ${.DEFAULT_GOAL} --no-print-directory
endif

bin-docker: bin build/linux-amd64/nebula build/linux-amd64/nebula-cert

smoke-docker: bin-docker
	cd .github/workflows/smoke/ && ./build.sh
	cd .github/workflows/smoke/ && ./smoke.sh

smoke-docker-race: BUILD_ARGS = -race
smoke-docker-race: smoke-docker

.FORCE:
.PHONY: e2e e2ev e2evv e2evvv e2evvvv test test-cov-html bench bench-cpu bench-cpu-long bin proto release service smoke-docker smoke-docker-race
.DEFAULT_GOAL := bin

# This must be the first line in Makefile
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir := $(dir $(mkfile_path))

BUILD=${mkfile_dir}build
GOBIN=${BUILD}/bin
GOSRC=${BUILD}/src
GOPROTO=${GOSRC}/flo_lb/proto
GO_PROTO_MODULE="github.com/FlorinBalint/flo_lb/proto"
BINARY=flo_load_balancer
CONFIG_FILE=configs/local.textproto

.PHONY: config_proto build run test clean

${GOPROTO}:
	mkdir -p ${GOPROTO}

${GOPROTO}/go.mod: ${GOPROTO}
		cd ${GOPROTO} && go mod init ${GO_PROTO_MODULE} && cd -

config_proto:
	protoc -I=${mkfile_dir}proto  --go_out=${GOPROTO} --go_opt=paths=source_relative \
		${mkfile_dir}proto/config.proto

${GOSRC}/${CONFIG_FILE}:
	mkdir -p $(dir ${GOSRC}/${CONFIG_FILE})
	cp ${mkfile_dir}${CONFIG_FILE} $(dir ${GOSRC}/${CONFIG_FILE})

build: ${GOPROTO}/go.mod config_proto
	go build ${LDFLAGS} -o ${GOBIN}/${BINARY} main.go

run: ${GOSRC}/${CONFIG_FILE}
	${GOBIN}/${BINARY} --config_file="${GOSRC}/${CONFIG_FILE}"

test:
	go test github.com/FlorinBalint/flo_lb/loadbalancer

clean:
	rm -rf ${BUILD}

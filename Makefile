# This must be the first line in Makefile
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir := $(dir $(mkfile_path))

BUILD=${mkfile_dir}build
GOBIN=${BUILD}/bin
GOPROTO=${BUILD}/src/proto
GO_PROTO_MODULE="github.com/FlorinBalint/flo_lb"
BINARY=flo_load_balancer
CONFIG_FILE=configs/prod.textproto

.PHONY: config_proto build run clean

${GOPROTO}/flo_lb:
	mkdir -p ${GOPROTO}/flo_lb

${GOPROTO}/flo_lb/go.mod: ${GOPROTO}/flo_lb
		cd ${GOPROTO}/flo_lb && go mod init ${GO_PROTO_MODULE} && cd -

config_proto:
	protoc -I=${mkfile_dir}proto  --go_out=${GOPROTO}/flo_lb --go_opt=paths=source_relative \
		--go-grpc_out=${GOPROTO}/flo_lb --go-grpc_opt=paths=source_relative \
		${mkfile_dir}proto/config.proto

${GOBIN}/${CONFIG_FILE}:
	mkdir -p $(dir ${GOBIN}/${CONFIG_FILE})
	cp ${mkfile_dir}${CONFIG_FILE} $(dir ${GOBIN}/${CONFIG_FILE})

build: ${GOPROTO}/flo_lb/go.mod config_proto ${GOBIN}/${CONFIG_FILE}
	go build ${LDFLAGS} -o ${GOBIN}/${BINARY} main.go

run:
	${GOBIN}/${BINARY} --config_file="${GOBIN}/${CONFIG_FILE}"

clean:
	rm -rf ${BUILD}
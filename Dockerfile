# syntax=docker/dockerfile:1

# Alpine for smaller footprint
FROM golang:1.18-alpine

RUN apk update && apk add --no-cache make protobuf-dev \
  && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
WORKDIR /app
ADD . ./
RUN make build
ENV CONFIG_FILE="./configs/prod.textproto"
ENV PORT "443"
ENV BINARY "/app/build/bin/flo_load_balancer"
EXPOSE ${PORT}
ENTRYPOINT [ "sh", "-c", "${BINARY} --config_file=${CONFIG_FILE}" ]

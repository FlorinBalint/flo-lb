# syntax=docker/dockerfile:1

# Alpine for smaller footprint
FROM golang:1.18-alpine

ENV CONFIG_FILE "./configs/docker.textproto"
ENV PORT "8080"
RUN apk update && apk add --no-cache make protobuf-dev \
  && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
WORKDIR /app
ADD . ./
RUN make build
EXPOSE 8080
CMD [ "sh", "-c", "/app/build/bin/flo_load_balancer --config_file=${CONFIG_FILE} --port=${PORT}" ]

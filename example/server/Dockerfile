# syntax=docker/dockerfile:1

# Alpine for smaller footprint
FROM golang:1.18-alpine

ENV PORT=8081
ENV SRV_NAME "Server"
WORKDIR /app
COPY ./main.go ./
EXPOSE ${PORT}
RUN go build -o server /app/main.go
CMD [ "sh", "-c", "/app/server --name=${SRV_NAME} --port=${PORT}" ]

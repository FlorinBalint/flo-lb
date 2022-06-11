Example of toy load balancer for educational purposes.

The current version round robins between hardcoded backends, without checking
for their health.

# Testing setup

Open 4 terminal windows and then run in each respectively:

## Terminal #1

Run the first test server backend on `localhost:8081`
```console
florinbalin@DESKTOP:flo-lb$ go run ./example/server/main.go --name=Server1 --port=8081
```

## Terminal #2
Run the second test server backend on `localhost:8082`
```console
florinbalin@DESKTOP:flo-lb$ go run ./example/server/main.go --name=Server2 --port=8082
```

## Terminal #3
Run the load balancer and make it point to the two servers:
```console
florinbalin@DESKTOP:flo-lb$ make build && make build
```

The load balancer will listen by default on port `:8080` 
and round robin requests to the two backends.
You can change the behaviour in the `configs/prod.textproto` config file.

## Terminal #4
Start doing `http` requests to the load balancer using curl:

```console
florinbalin@DESKTOP:~$ for i in {1..5}
do
curl http://localhost:8080/hello
done
```

Watch the logs of the servers, load balancers and the responses to
know what is happening.

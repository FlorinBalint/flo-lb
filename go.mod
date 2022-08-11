module github.com/FlorinBalint/flo_lb

go 1.18

require (
	github.com/FlorinBalint/flo_lb/proto v0.1.0
	github.com/google/go-cmp v0.5.8
	github.com/vine-io/vine v1.4.9
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d
	google.golang.org/protobuf v1.28.1
)

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

require (
	golang.org/x/exp v0.0.0-20220706164943-b4a6d9510983
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/text v0.3.7 // indirect
)

replace github.com/FlorinBalint/flo_lb/proto => ./build/src/flo_lb/proto

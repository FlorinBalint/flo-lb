module github.com/FlorinBalint/flo_lb

go 1.18

require (
	github.com/FlorinBalint/flo_lb/proto v0.1.0
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d
	google.golang.org/protobuf v1.28.0
)

require (
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2 // indirect
	golang.org/x/text v0.3.6 // indirect
)

replace github.com/FlorinBalint/flo_lb/proto => ./build/src/flo_lb/proto

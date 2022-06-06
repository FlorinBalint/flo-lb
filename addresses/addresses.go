package addresses

import (
	"flag"
	"strings"
)

func Flag(name string, defaultVal Addresses, description string) *Addresses {
	value := defaultVal
	flag.Var(&value, name, description)
	return &value
}

type Addresses []string

func (adrs *Addresses) String() string {
	return strings.Join(*adrs, ", ")
}

func (adrs *Addresses) Set(value string) error {
	if len(value) == 0 {
		return nil
	}

	vals := strings.Split(value, ",")
	*adrs = vals
	return nil
}

func (adrs *Addresses) Append(address string) {
	*adrs = append(*adrs, address)
}

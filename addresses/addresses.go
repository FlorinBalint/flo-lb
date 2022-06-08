package addresses

import (
	"flag"
	"strings"

	"net/url"
)

func Flag(name string, defaultVal Addresses, description string) *Addresses {
	value := defaultVal
	flag.Var(&value, name, description)
	return &value
}

type Addresses []*url.URL

func (adrs *Addresses) String() string {
	if len(*adrs) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString((*adrs)[0].String())

	for i := 1; i < len(*adrs); i++ {
		builder.WriteRune(',')
		builder.WriteString((*adrs)[i].String())
	}

	return builder.String()
}

func (adrs *Addresses) Set(value string) error {
	if len(value) == 0 {
		return nil
	}

	vals := strings.Split(value, ",")
	urls := make([]*url.URL, len(vals))

	for i := range vals {
		targetURL, err := url.Parse(vals[i])
		if err != nil {
			return err
		}
		urls[i] = targetURL
	}

	*adrs = urls
	return nil
}

func (adrs *Addresses) AppendString(address string) error {
	targetURL, err := url.Parse(address)
	if err != nil {
		return err
	}
	adrs.Append(targetURL)
	return nil
}

func (adrs *Addresses) Append(url *url.URL) {
	*adrs = append(*adrs, url)
}

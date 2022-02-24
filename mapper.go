package infector

import (
	"net/http"

	"google.golang.org/grpc/metadata"
)

func WrapMapper(container interface{}) Mapper {
	return Mapper{container}
}

type Mapper struct {
	container interface{}
}

func (ma *Mapper) Set(k, v string) {
	switch mapper := ma.container.(type) {
	case http.Header:
		mapper.Set(k, v)

	case metadata.MD:
		mapper.Set(k, v)

	case map[string]string:
		mapper[k] = v

	case map[string]interface{}:
		mapper[k] = v
	}
}

func (ma *Mapper) Get(k string) string {
	switch mapper := ma.container.(type) {
	case http.Header:
		return mapper.Get(k)

	case metadata.MD:
		list := mapper.Get(k)
		if len(list) == 0 {
			return ""
		}
		return list[0]

	case map[string]string:
		return mapper[k]

	case map[string]interface{}:
		v, ok := mapper[k]
		if !ok {
			return ""
		}
		return v.(string)
	}
	return ""
}

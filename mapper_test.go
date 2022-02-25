package infector

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestMapper(t *testing.T) {
	// header
	header := http.Header{}
	mapper := WrapMapper(header)
	mapper.Set("k1", "v1")
	assert.Equal(t, "v1", mapper.Get("k1"))
	assert.Equal(t, "v1", header.Get("k1"))

	// grpc metadata
	md := metadata.MD{}
	mapper = WrapMapper(md)
	mapper.Set("k1", "v1")
	assert.Equal(t, "v1", mapper.Get("k1"))
	assert.Equal(t, "v1", md.Get("k1")[0])

	// map
	cmap := map[string]string{}
	mapper = WrapMapper(cmap)
	mapper.Set("k1", "v1")
	assert.Equal(t, "v1", mapper.Get("k1"))
	assert.Equal(t, "v1", cmap["k1"])

	// map interface{}
	inmap := map[string]interface{}{}
	mapper = WrapMapper(inmap)
	mapper.Set("k1", "v1")
	assert.Equal(t, "v1", mapper.Get("k1"))
	assert.Equal(t, "v1", inmap["k1"].(string))
}

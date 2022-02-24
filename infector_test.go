package infector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToTime(t *testing.T) {
	var unix int64 = 1645540727000 // unit: ms
	ts := convTime(unix)
	assert.Equal(t, "2022-02-22 22:38:47 +0800 CST", ts.String())
}

package redis_test

import (
	"strings"
	"testing"

	redis "github.com/Kostaaa1/redis-clone/internal/resp"
	"github.com/stretchr/testify/require"
)

func TestResp_ReadBulk(t *testing.T) {
	t.Parallel()

	bulkStr := "$5\r\nworld\r\n"
	r := redis.NewReader(strings.NewReader(bulkStr))

	value, err := r.Read()
	require.NoError(t, err)
	require.Equal(t, value.Type, "bulk")
	require.Equal(t, len(value.Bulk), 5)
	require.Equal(t, value.Bulk, "world")
}

func TestResp_ReadArray(t *testing.T) {
	t.Parallel()

	bulkStr := "*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n"
	r := redis.NewReader(strings.NewReader(bulkStr))

	value, err := r.Read()
	require.NoError(t, err)
	require.Equal(t, value.Type, "array")
	require.Equal(t, len(value.Array), 2)
	require.Equal(t, value.Array[0].Bulk, "hello")
	require.Equal(t, value.Array[1].Bulk, "world")
}

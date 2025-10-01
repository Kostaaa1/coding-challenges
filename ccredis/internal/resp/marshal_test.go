package redis_test

import (
	"testing"

	redis "github.com/Kostaaa1/redis-clone/internal/resp"
	"github.com/stretchr/testify/require"
)

func TestResp_MarshalBulk(t *testing.T) {
	t.Parallel()

	bulkStr := []byte("$5\r\nhello\r\n")
	v := redis.Value{Type: "bulk", Bulk: "hello"}

	bytes := v.Marshal()
	require.Equal(t, bytes, bulkStr)
	require.Equal(t, len(bytes), len(bulkStr))
}

func TestResp_MarshalEmptyBulk(t *testing.T) {
	t.Parallel()

	bulkStr := []byte("$0\r\n\r\n")
	v := redis.Value{Type: "bulk", Bulk: ""}

	bytes := v.Marshal()
	require.Equal(t, bytes, bulkStr)
	require.Equal(t, len(bytes), len(bulkStr))
}

func TestResp_MarshalArray(t *testing.T) {
	t.Parallel()

	array := []byte("*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n")
	v := redis.Value{
		Type: "array",
		Array: []redis.Value{
			{Type: "bulk", Bulk: "hello"},
			{Type: "bulk", Bulk: "world"},
		},
	}
	bytes := v.Marshal()
	require.Equal(t, bytes, array)
	require.Equal(t, len(bytes), len(array))
}

func TestResp_MarshalString(t *testing.T) {
	t.Parallel()

	bulkStr := []byte("+OK\r\n")
	v := redis.Value{Type: "string", String: "OK"}

	bytes := v.Marshal()
	require.Equal(t, bytes, bulkStr)
	require.Equal(t, len(bytes), len(bulkStr))
}

func TestResp_MarshalNull(t *testing.T) {
	t.Parallel()

	bulkStr := []byte("$-1\r\n")
	v := redis.Value{Type: "null"}
	bytes := v.Marshal()
	require.Equal(t, bytes, bulkStr)
	require.Equal(t, len(bytes), len(bulkStr))
}

func TestResp_MarshalError(t *testing.T) {
	t.Parallel()

	errorStr := []byte("-Error message\r\n")
	v := redis.Value{Type: "error", String: "Error message"}
	bytes := v.Marshal()
	require.Equal(t, bytes, errorStr)
	require.Equal(t, len(bytes), len(errorStr))
}

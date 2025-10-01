package redis

type redisType int

const (
	REDIS_STRING redisType = iota
	REDIS_HASH
	REDIS_LIST
	REDIS_SET
	REDIS_ZSET
	REDIS_STREAM
)

func (t redisType) String() string {
	switch t {
	case REDIS_STRING:
		return "string"
	case REDIS_LIST:
		return "list"
	case REDIS_SET:
		return "set"
	case REDIS_HASH:
		return "hash"
	case REDIS_STREAM:
		return "stream"
	case REDIS_ZSET:
		return "zset"
	default:
		return "none"
	}
}

func TYPE(args []Value) Value {
	key := args[0].Bulk
	storeMu.RLock()
	obj, ok := store[key]
	storeMu.RUnlock()

	if ok {
		return Value{Type: "string", String: obj.itemType.String()}
	}
	return Value{Type: "string", String: "none"}
}

func errWrongType() Value {
	return errVal("WRONGTYPE Operation against a key holding the wrong kind of value")
}

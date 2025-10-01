package redis

import (
	"strconv"
)

type Value struct {
	Type   string
	Bulk   string
	Array  []Value
	String string
	Int    int
}

func (v Value) Marshal() []byte {
	switch v.Type {
	case "array":
		return v.marshalArray()
	case "bulk":
		return v.marshalBulk()
	case "error":
		return v.marshalError()
	case "null":
		return v.marshalNull()
	case "string":
		return v.marshalString()
	case "integer":
		return v.marshalInt()
	default:
		return []byte{}
	}
}

func (v Value) marshalInt() []byte {
	var bytes []byte
	bytes = append(bytes, INT)
	bytes = append(bytes, strconv.Itoa(v.Int)...)
	bytes = append(bytes, '\r', '\n')
	return bytes
}

func (v Value) marshalArray() []byte {
	var bytes []byte
	bytes = append(bytes, ARRAY)
	bytes = append(bytes, strconv.Itoa(len(v.Array))...)
	bytes = append(bytes, '\r', '\n')
	for _, v := range v.Array {
		bytes = append(bytes, v.Marshal()...)
	}
	// bytes = append(bytes, '\r', '\n')
	return bytes
}

func (v Value) marshalBulk() []byte {
	var bytes []byte
	bytes = append(bytes, BULK)

	if len(v.Bulk) == 0 {
		bytes = append(bytes, '0')
		bytes = append(bytes, '\r', '\n')
		bytes = append(bytes, '\r', '\n')
		return bytes
	}

	bytes = append(bytes, strconv.Itoa(len(v.Bulk))...)
	bytes = append(bytes, '\r', '\n')
	bytes = append(bytes, v.Bulk...)
	bytes = append(bytes, '\r', '\n')
	return bytes
}

func (v Value) marshalString() []byte {
	var bytes []byte
	bytes = append(bytes, STRING)
	bytes = append(bytes, v.String...)
	bytes = append(bytes, '\r', '\n')
	return bytes
}

func (v Value) marshalError() []byte {
	var bytes []byte
	bytes = append(bytes, ERROR)
	bytes = append(bytes, "ERR "...)
	bytes = append(bytes, v.String...)
	bytes = append(bytes, '\r', '\n')
	return bytes
}

func (v Value) marshalNull() []byte { return []byte("$-1\r\n") }

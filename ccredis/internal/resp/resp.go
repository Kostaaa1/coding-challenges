package redis

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
)

type Resp struct {
	reader *bufio.Reader
}

func NewReader(r io.Reader) *Resp {
	return &Resp{reader: bufio.NewReader(r)}
}

const (
	ARRAY  = '*'
	BULK   = '$'
	ERROR  = '-'
	STRING = '+'
	INT    = ':'
)

func (r *Resp) Read() (Value, error) {
	b, err := r.reader.ReadByte()
	if err != nil {
		return Value{}, err
	}

	switch b {
	case ARRAY:
		return r.readArray()
	case BULK:
		return r.readBulk()
	default:
		return Value{}, errors.New("unknown sign?")
	}
}

func (r *Resp) readLine() (n int, buf []byte, err error) {
	for {
		b, err := r.reader.ReadByte()
		if err != nil {
			return 0, nil, err
		}

		n++
		buf = append(buf, b)

		if len(buf) >= 2 && buf[len(buf)-2] == '\r' {
			break
		}
	}

	return n - 2, buf[:len(buf)-2], nil
}

func (r *Resp) readInt() (int, error) {
	_, line, err := r.readLine()
	if err != nil {
		return 0, err
	}
	i64, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	return int(i64), nil
}

func (r *Resp) readArray() (Value, error) {
	v := Value{
		Type: "array",
	}

	length, err := r.readInt()
	if err != nil {
		fmt.Println("error reading int: ", err)
		return v, err
	}

	v.Array = make([]Value, length)

	for i := range length {
		readval, err := r.Read()
		if err != nil {
			fmt.Println("error reading value: ", i, err)
			return v, err
		}
		v.Array[i] = readval
	}

	return v, nil
}

func (r *Resp) readBulk() (Value, error) {
	v := Value{}
	v.Type = "bulk"

	length, err := r.readInt()
	if err != nil {
		return v, err
	}

	buf := make([]byte, length)

	_, err = r.reader.Read(buf)
	if err != nil {
		return v, err
	}

	r.readLine()

	v.Bulk = string(buf)
	return v, nil
}

package redis

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RedisItem struct {
	value    interface{}
	itemType redisType
	ttl      time.Time
}

type HandlerFunc func(args []Value) Value

var (
	storeMu  sync.RWMutex
	store    = make(map[string]*RedisItem)
	Handlers = map[string]HandlerFunc{
		"MSET":     middleware(MSET),
		"SET":      middleware(SET),
		"GET":      middleware(GET),
		"DEL":      middleware(DEL),
		"TTL":      middleware(TTL),
		"TYPE":     middleware(TYPE),
		"KEYS":     middleware(KEYS),
		"EXPIRE":   middleware(EXPIRE),
		"PING":     PONG,
		"PERSIST":  PERSIST,
		"FLUSHALL": FLUSHALL,
	}
)

func EXPIRE(args []Value) Value {
	key := args[0].Bulk

	if len(args) < 2 {
		return errWrongArgs("expire")
	}

	expiry, err := strconv.Atoi(args[1].Bulk)
	if err != nil {
		return Value{}
	}
	ttl := time.Time.Add(time.Now(), time.Second*time.Duration(expiry))

	var nx, xx, gt, lt bool
	if len(args) > 3 {
		for _, arg := range args {
			switch arg.Bulk {
			case "NX":
				nx = true
			case "XX":
				xx = true
			case "GT":
				gt = true
			case "LT":
				lt = true
			}
		}
	}

	status := 0

	storeMu.Lock()
	if v, ok := store[key]; ok {
		shouldSet := false
		if nx && v.ttl.IsZero() {
			shouldSet = true
		} else if !v.ttl.IsZero() {
			if xx {
				shouldSet = true
			}
			if gt && ttl.After(v.ttl) {
				shouldSet = true
			}
			if lt && ttl.Before(v.ttl) {
				shouldSet = true
			}
		} else {
			shouldSet = true
		}
		if shouldSet {
			status = 1
			v.ttl = ttl
		}
	}
	storeMu.Unlock()

	return Value{Type: "integer", Int: status}
}

func PERSIST(args []Value) Value {
	key := args[0].Bulk
	if v, ok := store[key]; ok {
		storeMu.Lock()
		v.ttl = time.Time{}
		storeMu.Unlock()
	}
	return ok()
}

func KEYS(args []Value) Value {
	typ := args[0].Bulk

	storeMu.RLock()
	defer storeMu.RUnlock()

	v := Value{Type: "array", Array: []Value{}}

	for _, item := range store {
		switch typ {
		case "*":
			if itemVal, ok := item.value.(string); ok {
				newVal := Value{Type: "bulk", Bulk: itemVal}
				v.Array = append(v.Array, newVal)
			}
		}
	}

	return v
}

func FLUSHALL(args []Value) Value {
	storeMu.Lock()
	store = make(map[string]*RedisItem)
	storeMu.Unlock()
	return ok()
}

func TTL(args []Value) Value {
	key := args[0].Bulk

	storeMu.Lock()
	defer storeMu.Unlock()

	obj, ok := store[key]
	if !ok {
		return intVal(-2)
	}

	if isExpired(obj.ttl) {
		delete(store, key)
		return intVal(-2)
	}

	if obj.ttl.IsZero() {
		return intVal(-1)
	}

	return intVal(int(time.Until(obj.ttl).Seconds()))
}

func PONG(args []Value) Value {
	return strVal("PONG")
}

func errWrongArgs(cmd string) Value {
	msg := fmt.Sprintf("wrong number of arguments for '%s' command", cmd)
	return errVal(msg)
}

func UnknownCmd(cmd string, args []Value) Value {
	cmds := make([]string, len(args))
	for i, arg := range args {
		cmds[i] = fmt.Sprintf("'%s'", arg.Bulk)
	}
	msg := fmt.Sprintf("unknown command '%s', with args beginning with: %s", cmd, strings.Join(cmds, " "))
	return errVal(msg)
}

func addTTL(strVal string, dur time.Duration) (time.Time, error) {
	ttl, err := strconv.Atoi(strVal)
	if err != nil {
		return time.Time{}, errors.New("value is not an integer or out of range")
	}
	if ttl < 0 {
		return time.Time{}, errors.New("invalid expire time in 'set' command")
	}
	return time.Now().Add(time.Duration(ttl) * dur), nil
}

func isExpired(ttl time.Time) bool {
	if !ttl.IsZero() && time.Now().After(ttl) {
		return true
	}
	return false
}

package redis

import (
	"time"
)

func DEL(args []Value) Value {
	storeMu.Lock()
	defer storeMu.Unlock()

	c := 0
	for _, opt := range args {
		if _, ok := store[opt.Bulk]; ok {
			c++
			delete(store, opt.Bulk)
		}
	}

	return intVal(c)
}

func GET(args []Value) Value {
	key := args[0].Bulk

	storeMu.RLock()
	obj, ok := store[key]
	expired := ok && isExpired(obj.ttl)
	val := obj.value
	storeMu.RUnlock()

	if !ok || expired {
		if expired {
			storeMu.Lock()
			if val2, ok := store[key]; ok && isExpired(val2.ttl) {
				delete(store, key)
			}
			storeMu.Unlock()
		}
		return nullVal()
	}

	return bulkVal(val.(string))
}

func MSET(args []Value) Value {
	if len(args)%2 != 0 {
		return errWrongArgs("mset")
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	for i := 0; i < len(args); i++ {
		key := args[i].Bulk
		val := args[i+1].Bulk
		store[key] = &RedisItem{itemType: REDIS_STRING, value: val}
		i++
	}

	return ok()
}

// Options
// EX - seconds
// PX - milliseconds
// [TODO] EXAT - set specific unix time, in seconds - positive int
// [TODO] PXAT - set specific unix time, in milliseconds - positive int
// NX - only set the key if it does not exist
// XX - only set the key if it exists
// GET - return the old string or nil if key did not exist.
// KEEPTTL - Retain the TTL
func SET(args []Value) Value {
	key := args[0].Bulk
	value := args[1].Bulk
	newval := &RedisItem{itemType: REDIS_STRING, value: value, ttl: time.Time{}}

	var nx, xx, get, keepttl bool
	var ttlOptCount int

	opts := args[2:]

	for i, opt := range opts {
		switch opt.Bulk {
		case "EX", "PX":
			ttlOptCount++
			if ttlOptCount > 1 || i+1 >= len(opts) {
				return syntaxErr()
			}

			var dur time.Duration
			if opt.Bulk == "EX" {
				dur = time.Second
			} else {
				dur = time.Millisecond
			}

			ttl, err := addTTL(opts[i+1].Bulk, dur)
			if err != nil {
				return errVal(err.Error())
			}
			newval.ttl = ttl
		case "KEEPTTL":
			ttlOptCount++
			if ttlOptCount > 1 {
				return syntaxErr()
			}
			keepttl = true
		// case "EXAT":
		// case "PXAT":
		case "NX":
			nx = true
		case "XX":
			xx = true
		case "GET":
			get = true
		default:
			return syntaxErr()
		}
	}

	if xx && nx {
		return syntaxErr()
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	val, exists := store[key]

	if nx && exists || xx && !exists {
		return nullVal()
	}

	if exists && keepttl {
		newval.ttl = val.ttl
	}

	store[key] = newval

	if get && exists {
		return bulkVal(val.value.(string))
	}

	return ok()
}

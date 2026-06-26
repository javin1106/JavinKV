package core

import (
	"errors"
	"io"
	"log"
	"strconv"
	"time"
)

var RESP_NIL []byte = []byte("$-1\r\n")

func evalPING(args []string, c io.ReadWriter) error {
	var b []byte
	if len(args) >= 2 {
		return errors.New("ERR wrong number of arguments for 'ping' command")
	}

	if len(args) == 0 {
		// PING with no arguments returns the standard simple-string reply.
		b = Encode("PONG", true)
	} else {
		// PING with one argument echoes that value back as a bulk string.
		b = Encode(args[0], false)
	}

	// Write the encoded RESP reply back to the connected client.
	_, err := c.Write(b)
	return err
}

func evalSET(args []string, c io.ReadWriter) error {
	if len(args) <= 1 {
		return errors.New("(error) ERR wrong number of arguments for 'set' command")
	}

	var key, value string
	var exDurationMs int64 = -1 // never expire

	key, value = args[0], args[1]
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "EX", "ex":
			i++
			if i == len(args) {
				return errors.New("(error) ERR syntax error")
			}
			exDurationSec, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return errors.New("(error) ERR value is not an integer or out of range")
			}
			if exDurationSec <= 0 {
				return errors.New("(error) ERR invalid expire time in 'set' command")
			}
			exDurationMs = exDurationSec * 1000 // in Redis, time is by default in ms

		default:
			return errors.New("(error) ERR syntax error")
		}
	}
	Put(key, NewObj(value, exDurationMs))
	c.Write([]byte("+OK\r\n"))
	return nil
}

func evalGET(args []string, c io.ReadWriter) error {
	if len(args) != 1 {
		return errors.New("(error) ERR wrong number of arguments for 'get' command")
	}

	var key string = args[0]

	// Get the key from the hash table
	obj := Get(key)

	// if key does not exist, return RESP encoded nil
	if obj == nil {
		c.Write(RESP_NIL)
		return nil
	}

	// if key already expired then return nil
	if obj.ExpiresAt != -1 && obj.ExpiresAt <= time.Now().UnixMilli() {
		Del(key)
		c.Write(RESP_NIL)
		return nil
	}

	// return the RESP encoded value
	c.Write(Encode(obj.Value, false))
	return nil
}

func evalTTL(args []string, c io.ReadWriter) error {
	if len(args) != 1 {
		return errors.New("(error) ERR wrong number of arguments for 'ttl' command")
	}

	var key string = args[0]

	obj := Get(key)

	// if key does not exist, return RESP encoded -2 denoting key does not exist
	if obj == nil {
		c.Write([]byte(":-2\r\n"))
		return nil
	}

	durationMs := obj.ExpiresAt - time.Now().UnixMilli()

	// if key expired i.e. key does not exist hence return -2
	if durationMs < 0 {
		Del(key)
		c.Write([]byte(":-2\r\n"))
		return nil
	}

	// if object exist, but no expiration is set on it then send -1
	if obj.ExpiresAt == -1 {
		c.Write([]byte(":-1\r\n"))
		return nil
	}

	// compute the time remaining for the key to expire and
	// return the RESP encoded form of it
	c.Write(Encode(int64(durationMs/1000), false))
	return nil
}

func evalDEL(args []string, c io.ReadWriter) error {
	if len(args) == 0 {
		return errors.New("(error)ERR wrong number of arguments for 'del' command")
	}
	var countDeleted int = 0
	for _, key := range args {
		obj := Get(key)
		if obj != nil && obj.ExpiresAt != -1 && obj.ExpiresAt <= time.Now().UnixMilli() {
			Del(key)
			continue
		}
		if ok := Del(key); ok {
			countDeleted++
		}
	}
	c.Write(Encode(countDeleted, false))
	return nil
}

func evalEXPIRE(args []string, c io.ReadWriter) error {
	if len(args) != 2 {
		return errors.New("(error)ERR wrong number of arguments for 'expire' command")
	}
	var key string = args[0]
	exDurationSec, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return errors.New("(error) ERR value is not an integer or out of range")
	}
	if exDurationSec <= 0 {
		return errors.New("(error) ERR invalid expire time in 'expire' command")
	}
	obj := Get(key)

	// unsuccessful -> doesn't exits, timeout not set
	if obj == nil {
		c.Write([]byte(":0\r\n"))
		return nil
	}
	if obj.ExpiresAt != -1 && obj.ExpiresAt <= time.Now().UnixMilli() {
		Del(key)
		c.Write([]byte(":0\r\n"))
		return nil
	}

	obj.ExpiresAt = time.Now().UnixMilli() + exDurationSec*1000
	c.Write([]byte(":1\r\n"))
	return nil
}

func EvalAndRespond(cmd *RedisCmd, c io.ReadWriter) error {
	log.Println("command", cmd.Cmd)
	// Dispatch the parsed command to the matching handler.
	switch cmd.Cmd {
	case "PING":
		return evalPING(cmd.Args, c)
	case "SET":
		return evalSET(cmd.Args, c)
	case "GET":
		return evalGET(cmd.Args, c)
	case "TTL":
		return evalTTL(cmd.Args, c)
	case "DEL":
		return evalDEL(cmd.Args, c)
	case "EXPIRE":
		return evalEXPIRE(cmd.Args, c)
	default:
		return errors.New("(error) ERR unknown command")
	}
}

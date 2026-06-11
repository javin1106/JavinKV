package core

import (
	"errors"
	"log"
	"net"
)

func evalPING(args []string, c net.Conn) error {
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

func EvalAndRespond(cmd *RedisCmd, c net.Conn) error {
	log.Println("command", cmd.Cmd)
	// Dispatch the parsed command to the matching handler.
	switch cmd.Cmd {
	case "PING":
		return evalPING(cmd.Args, c)
	default:
		return evalPING(cmd.Args, c) // Just for now, will implement further commands and change default
	}
}

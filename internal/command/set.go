package command

import (
	"time"

	"github.com/nalgeon/redka/internal/core"
	"github.com/nalgeon/redka/internal/parser"
)

// Set sets the string value of a key.
// The key is created if it doesn't exist.
// SET key value [NX | XX] [GET] [EX seconds | PX milliseconds | EXAT unix-time-seconds | PXAT unix-time-milliseconds | KEEPTTL]
// https://redis.io/commands/set
type Set struct {
	baseCmd
	key     string
	value   []byte
	ifNX    bool
	ifXX    bool
	get     bool
	ttl     time.Duration
	at      time.Time
	keepTTL bool
}

func parseSet(b baseCmd) (*Set, error) {
	cmd := &Set{baseCmd: b}

	// Parse the command arguments.
	var ttlSec, ttlMs, atSec, atMs int
	err := parser.New(
		parser.String(&cmd.key),
		parser.Bytes(&cmd.value),
		parser.OneOf(
			parser.Flag("nx", &cmd.ifNX),
			parser.Flag("xx", &cmd.ifXX),
		),
		parser.Flag("get", &cmd.get),
		parser.OneOf(
			parser.Named("ex", parser.Int(&ttlSec)),
			parser.Named("px", parser.Int(&ttlMs)),
			parser.Named("exat", parser.Int(&atSec)),
			parser.Named("pxat", parser.Int(&atMs)),
			parser.Flag("keepttl", &cmd.keepTTL),
		),
	).Required(2).Run(cmd.args)
	if err != nil {
		return cmd, err
	}

	// Set the expiration time.
	if ttlSec > 0 {
		cmd.ttl = time.Duration(ttlSec) * time.Second
	} else if ttlMs > 0 {
		cmd.ttl = time.Duration(ttlMs) * time.Millisecond
	} else if atSec > 0 {
		cmd.at = time.Unix(int64(atSec), 0)
	} else if atMs > 0 {
		cmd.at = time.Unix(0, int64(atMs)*int64(time.Millisecond))
	}
	if cmd.ttl < 0 {
		return cmd, ErrInvalidExpireTime
	}

	return cmd, nil
}

func (cmd *Set) Run(w Writer, red Redka) (any, error) {
	// Build and run the command.
	op := red.Str().SetWith(cmd.key, cmd.value)
	if cmd.ifXX {
		op = op.IfExists()
	} else if cmd.ifNX {
		op = op.IfNotExists()
	}
	if cmd.ttl > 0 {
		op = op.TTL(cmd.ttl)
	} else if !cmd.at.IsZero() {
		op = op.At(cmd.at)
	} else if cmd.keepTTL {
		op = op.KeepTTL()
	}
	out, err := op.Run()

	// Determine the output status.
	var ok bool
	if cmd.ifXX {
		ok = out.Updated
	} else if cmd.ifNX {
		ok = out.Created
	} else {
		ok = err == nil
	}

	// Write the output.
	if err != nil {
		w.WriteError(cmd.Error(err))
		return nil, err
	}

	if cmd.get {
		// GET given: The key didn't exist before the SET.
		if !out.Prev.Exists() {
			w.WriteNull()
			return core.Value(nil), nil
		}
		// GET given: The previous value of the key.
		w.WriteBulk(out.Prev)
		return out.Prev, nil
	} else {
		// GET not given: Operation was aborted (conflict with one of the XX/NX options).
		if !ok {
			w.WriteNull()
			return false, nil
		}
		// GET not given: The key was set.
		w.WriteString("OK")
		return true, nil
	}
}

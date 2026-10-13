package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/dejitarudemon/ignicula-go/client"
	"github.com/dejitarudemon/ignicula-go/configs"
	"github.com/dejitarudemon/ignicula-go/logger/implemented"
	"github.com/dejitarudemon/ignicula-go/values"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:16000", "server host:port")
	login := flag.String("login", "user", "handshake login")
	password := flag.String("password", "", "handshake password (Argon2id-hashed before send)")
	anon := flag.Bool("anon", false, "anonymous handshake (empty login, zero hash)")
	flag.Parse()

	cfg := configs.DefaultConnectionConfig()
	cfg.Auth = configs.AuthConfig{
		Login:     *login,
		Password:  *password,
		UseAnonim: *anon,
	}

	log := implemented.NewText(slog.LevelDebug)
	cli, err := client.Dial(*addr, cfg, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial %s: %v\n", *addr, err)
		os.Exit(1)
	}
	defer func() {
		if err := cli.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close: %v\n", err)
		}
	}()

	fmt.Printf("connected to %s\n", *addr)
	printCommands()

	lines := make(chan string)
	go func() {
		in := bufio.NewScanner(os.Stdin)
		for in.Scan() {
			lines <- in.Text()
		}
		close(lines)
		if err := in.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "stdin: %v\n", err)
		}
	}()

	for {
		fmt.Print("> ")
		select {
		case <-cli.Done():
			if err := cli.Err(); err != nil && !errors.Is(err, client.ErrClosed) {
				fmt.Fprintf(os.Stderr, "disconnected: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("connection closed")
			return

		case line, ok := <-lines:
			if !ok {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if !runCommand(cli, line) {
				return
			}
		}
	}
}

// runCommand handles one interactive line. It returns false when the session
// should exit (quit).
func runCommand(cli *client.Ignicula, line string) bool {
	cmd, arg, _ := strings.Cut(line, " ")
	arg = strings.TrimSpace(arg)
	switch strings.ToLower(cmd) {
	case "ping", "p":
		start := time.Now()
		err := cli.Ping(context.Background())
		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("ping: %s (%s)\n", err, elapsed.Round(time.Microsecond))
			return true
		}
		fmt.Printf("pong (%s)\n", elapsed.Round(time.Microsecond))

	case "read", "r":
		if arg == "" {
			fmt.Println("usage: read <key>")
			return true
		}
		start := time.Now()
		val, err := cli.Read(context.Background(), []byte(arg))
		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("read: %s (%s)\n", err, elapsed.Round(time.Microsecond))
			return true
		}
		fmt.Printf("%s (%s)\n", formatValue(val), elapsed.Round(time.Microsecond))

	case "write", "w":
		key, rest, ok := strings.Cut(arg, " ")
		key, rest = strings.TrimSpace(key), strings.TrimSpace(rest)
		raw, typName, typ := parseWriteArgs(rest)
		if !ok || key == "" || raw == "" {
			fmt.Println("usage: write <key> <value> [bytes|string|int64|uint64|float64|json]")
			return true
		}
		val, err := values.New(typ, []byte(raw))
		if err != nil {
			fmt.Printf("write: %s\n", err)
			return true
		}
		if typName == "" {
			typName = strings.ToLower(typ.String())
		}
		start := time.Now()
		err = cli.Write(context.Background(), []byte(key), val)
		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("write: %s (%s)\n", err, elapsed.Round(time.Microsecond))
			return true
		}
		fmt.Printf("ok type=%s (%s)\n", typName, elapsed.Round(time.Microsecond))

	case "writetyped", "wt":
		key, rest, ok := strings.Cut(arg, " ")
		key, rest = strings.TrimSpace(key), strings.TrimSpace(rest)
		if !ok || key == "" || rest == "" {
			fmt.Println("usage: writetyped <key> <elemType> <elem>...")
			return true
		}
		val, typName, err := parseTypedArrayArgs(rest)
		if err != nil {
			fmt.Printf("writetyped: %s\n", err)
			return true
		}
		start := time.Now()
		err = cli.Write(context.Background(), []byte(key), val)
		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("writetyped: %s (%s)\n", err, elapsed.Round(time.Microsecond))
			return true
		}
		fmt.Printf("ok type=%s (%s)\n", typName, elapsed.Round(time.Microsecond))

	case "writeuntyped", "wu":
		key, rest, ok := strings.Cut(arg, " ")
		key, rest = strings.TrimSpace(key), strings.TrimSpace(rest)
		if !ok || key == "" {
			fmt.Println("usage: writeuntyped <key> <type:value>...")
			return true
		}
		val, typName, err := parseUntypedArrayArgs(rest)
		if err != nil {
			fmt.Printf("writeuntyped: %s\n", err)
			return true
		}
		start := time.Now()
		err = cli.Write(context.Background(), []byte(key), val)
		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("writeuntyped: %s (%s)\n", err, elapsed.Round(time.Microsecond))
			return true
		}
		fmt.Printf("ok type=%s (%s)\n", typName, elapsed.Round(time.Microsecond))

	case "delete", "del", "d":
		if arg == "" {
			fmt.Println("usage: delete <key>")
			return true
		}
		start := time.Now()
		err := cli.Delete(context.Background(), []byte(arg))
		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("delete: %s (%s)\n", err, elapsed.Round(time.Microsecond))
			return true
		}
		fmt.Printf("ok (%s)\n", elapsed.Round(time.Microsecond))

	case "help", "h", "?":
		printCommands()

	case "quit", "exit", "q":
		return false

	default:
		fmt.Printf("unknown command %q (try help)\n", line)
	}
	return true
}

func printCommands() {
	fmt.Println("commands:")
	fmt.Println("  ping                                 send a Ping and wait for answer")
	fmt.Println("  read <key>                           read a value by key")
	fmt.Println("  write <key> <value> [type]            scalar (bytes|string|int64|uint64|float64|json; default bytes)")
	fmt.Println("  writetyped <key> <elemType> <elem>... typed array (alias: wt)")
	fmt.Println("  writeuntyped <key> <type:value>...    untyped array (alias: wu)")
	fmt.Println("  delete <key>                         delete a value by key")
	fmt.Println("  help                                 show this command list")
	fmt.Println("  quit                                 close connection and exit")
}

func formatValue(val values.Value) string {
	if val == nil {
		return "value=<nil>"
	}
	switch v := val.(type) {
	case values.StringValue:
		return fmt.Sprintf("string %q", string(v))
	case values.BytesValue:
		return fmt.Sprintf("bytes %q", string(v))
	case values.Int64Value:
		return fmt.Sprintf("int64 %d", int64(v))
	case values.Uint64Value:
		return fmt.Sprintf("uint64 %d", uint64(v))
	case values.Float64Value:
		return fmt.Sprintf("float64 %g", float64(v))
	case values.JSONValue:
		return fmt.Sprintf("json %s", string(v))
	case values.TypedArrayValue:
		parts := make([]string, len(v.Elems))
		for i, e := range v.Elems {
			parts[i] = formatValue(e)
		}
		return fmt.Sprintf("typed_array[%s] [%s]", v.ElemType, strings.Join(parts, ", "))
	case values.UntypedArrayValue:
		parts := make([]string, len(v.Elems))
		for i, e := range v.Elems {
			parts[i] = formatValue(e)
		}
		return fmt.Sprintf("untyped_array [%s]", strings.Join(parts, ", "))
	default:
		return fmt.Sprintf("%T=%v", val, val)
	}
}

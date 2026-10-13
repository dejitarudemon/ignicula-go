package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/dejitarudemon/ignicula-go/client"
	"github.com/dejitarudemon/ignicula-go/configs"
	"github.com/dejitarudemon/ignicula-go/internal/transport"
	"github.com/dejitarudemon/ignicula-go/logger/implemented"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:16000", "server host:port")
	login := flag.String("login", "user", "handshake login")
	hashHex := flag.String("hash", "", "auth hash as 64 hex chars (default: 32 zero bytes)")
	flag.Parse()

	hash, err := parseHash(*hashHex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash: %v\n", err)
		os.Exit(2)
	}

	cfg := configs.DefaultConnectionConfig()
	cfg.Auth = configs.AuthConfig{
		Login: *login,
		Hash:  hash,
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
	fmt.Println("commands: ping | help | quit")

	if err := waitReady(cli, 15*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "activate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ready")

	in := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !in.Scan() {
			break
		}
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}

		cmd, _, _ := strings.Cut(line, " ")
		switch strings.ToLower(cmd) {
		case "ping", "p":
			start := time.Now()
			err := cli.Ping(context.Background())
			elapsed := time.Since(start)
			if err != nil {
				fmt.Printf("ping: %v (%s)\n", err, elapsed.Round(time.Microsecond))
				continue
			}
			fmt.Printf("pong (%s)\n", elapsed.Round(time.Microsecond))

		case "help", "h", "?":
			fmt.Println("  ping  send a Ping and wait for answer")
			fmt.Println("  quit  close connection and exit")

		case "quit", "exit", "q":
			return

		default:
			fmt.Printf("unknown command %q (try help)\n", line)
		}
	}
	if err := in.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "stdin: %v\n", err)
		os.Exit(1)
	}
}

func parseHash(s string) ([32]byte, error) {
	var out [32]byte
	if s == "" {
		return out, nil
	}
	raw, err := hex.DecodeString(s)
	if err != nil {
		return out, err
	}
	if len(raw) != 32 {
		return out, fmt.Errorf("want 32 bytes, got %d", len(raw))
	}
	copy(out[:], raw)
	return out, nil
}

func waitReady(cli *client.Ignicula, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		err := cli.Ping(context.Background())
		if err == nil {
			return nil
		}
		last = err
		// Keep retrying while activation is in flight; any other error is also
		// retried briefly in case Start has not stored it yet.
		if !errors.Is(err, transport.ErrNoActiveVersion) && time.Since(deadline.Add(-timeout)) > time.Second {
			return err
		}
		time.Sleep(20 * time.Millisecond)
	}
	if last == nil {
		return context.DeadlineExceeded
	}
	return last
}

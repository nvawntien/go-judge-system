// bootstrap-sessions prepares local cookie files through AstraCode's public API.
package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/bootstrap"

	"golang.org/x/sys/unix"
)

const maxPasswordBytes = 4096

var errInteractiveTTY = errors.New("interactive password input requires a terminal; use --password-file")

type options struct {
	baseURL      string
	start        int
	count        int
	output       string
	replace      bool
	allowRemote  bool
	confirmHost  string
	passwordFile string
	loginDelay   time.Duration
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "bootstrap-sessions:", safeError(err))
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdin *os.File, stdout, stderr io.Writer) error {
	opts, err := parse(args, stderr)
	if err != nil {
		return err
	}
	base, err := url.Parse(opts.baseURL)
	if err != nil {
		return bootstrap.ErrTargetUnsafe
	}
	if err := bootstrap.ValidateTarget(base, opts.allowRemote, opts.confirmHost); err != nil {
		return err
	}
	if _, err := bootstrap.Identities(opts.start, opts.count); err != nil {
		return err
	}
	if !bootstrap.SafePath(opts.output) {
		return errors.New("unsafe output path")
	}
	password, err := acquirePassword(ctx, stdin, stderr, opts.passwordFile)
	if err != nil {
		return err
	}
	defer clear(password)
	file, err := bootstrap.Run(ctx, bootstrap.Options{BaseURL: base, AllowRemote: opts.allowRemote, ConfirmTargetHost: opts.confirmHost, Start: opts.start, Count: opts.count, Password: password, Output: opts.output, Replace: opts.replace, LoginDelay: opts.loginDelay})
	if err != nil {
		return err
	}
	info, err := os.Stat(opts.output)
	if err != nil {
		return errors.New("inspect credential output")
	}
	fmt.Fprintf(stdout, "Bootstrapped sessions: %d\nValidated sessions: %d\nOutput: %s\nMode: %04o\n", len(file.Users), len(file.Users), opts.output, info.Mode().Perm())
	return nil
}

func parse(args []string, stderr io.Writer) (options, error) {
	var opts options
	fs := flag.NewFlagSet("bootstrap-sessions", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&opts.baseURL, "base-url", "", "public AstraCode gateway base URL")
	fs.IntVar(&opts.start, "start", 1, "first benchmark sequence (1..100)")
	fs.IntVar(&opts.count, "count", 50, "number of benchmark users")
	fs.StringVar(&opts.output, "output", "users.local.json", "local credential output path")
	fs.BoolVar(&opts.replace, "replace", false, "atomically replace an existing local credential file")
	fs.BoolVar(&opts.allowRemote, "allow-remote", false, "allow confirmed non-loopback HTTPS target")
	fs.StringVar(&opts.confirmHost, "confirm-target-host", "", "exact non-loopback target hostname")
	fs.StringVar(&opts.passwordFile, "password-file", "", "secure one-password file")
	fs.DurationVar(&opts.loginDelay, "login-delay", 50*time.Millisecond, "sequential delay between successful logins")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if fs.NArg() != 0 || opts.baseURL == "" || opts.loginDelay < 0 {
		return options{}, errors.New("invalid bootstrap arguments")
	}
	return opts, nil
}

func acquirePassword(ctx context.Context, stdin *os.File, stderr io.Writer, passwordFile string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if passwordFile != "" {
		return readPasswordFile(passwordFile)
	}
	fmt.Fprint(stderr, "Benchmark password: ")
	reader := bufio.NewReaderSize(stdin, maxPasswordBytes+2)
	password, err := readNoEcho(ctx, stdin, reader)
	fmt.Fprintln(stderr)
	if err != nil {
		return nil, err
	}
	return password, nil
}

func readNoEcho(ctx context.Context, stdin *os.File, reader *bufio.Reader) ([]byte, error) {
	fd := int(stdin.Fd())
	state, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, errInteractiveTTY
	}
	noEcho := *state
	noEcho.Lflag &^= unix.ECHO
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &noEcho); err != nil {
		return nil, errors.New("disable terminal password echo")
	}
	var once sync.Once
	restore := func() { once.Do(func() { _ = unix.IoctlSetTermios(fd, unix.TCSETS, state) }) }
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			restore()
			_ = stdin.Close()
		case <-done:
		}
	}()
	defer close(done)
	defer restore()
	password, err := reader.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) || len(password) > maxPasswordBytes+1 {
		clear(password)
		return nil, errors.New("benchmark password is too large")
	}
	if ctx.Err() != nil {
		clear(password)
		return nil, ctx.Err()
	}
	if err != nil && len(password) == 0 {
		return nil, errors.New("read benchmark password")
	}
	password = bytes.TrimSuffix(password, []byte("\n"))
	password = bytes.TrimSuffix(password, []byte("\r"))
	if len(password) == 0 {
		clear(password)
		return nil, errors.New("benchmark password must not be empty")
	}
	return password, nil
}

func readPasswordFile(path string) ([]byte, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, errors.New("read benchmark password file")
	}
	f := os.NewFile(uintptr(fd), "benchmark-password-file")
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("benchmark password file must be a secure regular file")
	}
	contents, err := io.ReadAll(io.LimitReader(f, maxPasswordBytes+2))
	if err != nil {
		return nil, errors.New("read benchmark password file")
	}
	if len(contents) > maxPasswordBytes+1 {
		clear(contents)
		return nil, errors.New("benchmark password file is too large")
	}
	if bytes.HasSuffix(contents, []byte("\n")) {
		contents = contents[:len(contents)-1]
		contents = bytes.TrimSuffix(contents, []byte("\r"))
	}
	if len(contents) == 0 || len(contents) > maxPasswordBytes || bytes.ContainsAny(contents, "\r\n") {
		clear(contents)
		return nil, errors.New("benchmark password file must contain one password")
	}
	return contents, nil
}

func safeError(err error) string {
	switch {
	case errors.Is(err, bootstrap.ErrInvalidRange):
		return "start/count must select benchmark accounts 001 through 100"
	case errors.Is(err, bootstrap.ErrTargetUnsafe):
		return "target must be loopback HTTP(S), or confirmed HTTPS remote"
	case errors.Is(err, errInteractiveTTY):
		return errInteractiveTTY.Error()
	case errors.Is(err, context.Canceled):
		return "bootstrap cancelled"
	default:
		return "safe bootstrap failure; review target, local files, and account state"
	}
}

func clear(value []byte) {
	for index := range value {
		value[index] = 0
	}
}

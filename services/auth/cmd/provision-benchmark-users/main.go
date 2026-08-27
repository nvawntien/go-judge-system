// provision-benchmark-users creates only the fixed benchmark fixture pool.
// It deliberately has no HTTP, token, mail, Redis, or migration dependencies.
package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"go-judge-system/pkg/config"
	"go-judge-system/pkg/database"
	"go-judge-system/services/auth/internal/adapter/outbound/persistence/postgres"
	"go-judge-system/services/auth/internal/adapter/outbound/security"
	benchmark "go-judge-system/services/auth/internal/application/usecase/benchmark"

	"golang.org/x/sys/unix"
)

const maxPasswordBytes = 4096

var errInteractivePasswordTTY = errors.New("interactive password input requires a terminal; use --password-file")

var (
	version   = "dev"
	buildTime = "unknown"
	commitSHA = "unknown"
)

type options struct {
	configDir      string
	start          int
	count          int
	apply          bool
	confirm        string
	passwordFile   string
	rotatePassword bool
	dryRun         bool
	showVersion    bool
}

type commandRuntime struct {
	target      string
	provisioner *benchmark.Provisioner
	close       func()
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "provisioning failed:", safeError(err))
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdin *os.File, stdout, stderr io.Writer) error {
	return runWithRuntime(ctx, args, stdin, stdout, stderr, openCommandRuntime)
}

func runWithRuntime(ctx context.Context, args []string, stdin *os.File, stdout, stderr io.Writer, openRuntime func(string) (commandRuntime, error)) error {
	opts, err := parseOptions(args, stderr)
	if err != nil {
		return err
	}
	if opts.showVersion {
		fmt.Fprintf(stdout, "version=%s commit=%s build_time=%s\n", version, commitSHA, buildTime)
		return nil
	}
	identities, err := benchmark.Identities(opts.start, opts.count)
	if err != nil {
		return err
	}
	runtime, err := openRuntime(opts.configDir)
	if err != nil {
		return err
	}
	defer runtime.close()
	confirmation := confirmationPhrase(identities, runtime.target, opts.rotatePassword)
	started := time.Now()
	plan, err := planForMode(runtime.provisioner, ctx, opts)
	if err != nil {
		return err
	}
	if !opts.apply {
		printResult(stdout, false, opts.rotatePassword, runtime.target, identities, confirmation, plan, time.Since(started))
		return nil
	}
	if plan.Conflicts != 0 {
		printResult(stdout, true, opts.rotatePassword, runtime.target, identities, confirmation, plan, time.Since(started))
		return benchmark.ErrConflicts
	}
	if err := validateApplyConfirmation(opts, confirmation, stderr); err != nil {
		return err
	}

	password, err := acquirePassword(ctx, stdin, stderr, opts.passwordFile)
	if err != nil {
		return err
	}
	defer clear(password)
	req := benchmark.Request{Start: opts.start, Count: opts.count, Apply: true, Password: password, Progress: progressReporter(stdout)}
	var result benchmark.Result
	var executeErr error
	if opts.rotatePassword {
		result, executeErr = runtime.provisioner.RotatePassword(ctx, req)
	} else {
		result, executeErr = runtime.provisioner.Execute(ctx, req)
	}
	printResult(stdout, true, opts.rotatePassword, runtime.target, identities, confirmation, result, time.Since(started))
	if executeErr != nil {
		return executeErr
	}
	return nil
}

func openCommandRuntime(configDir string) (commandRuntime, error) {
	cfg, err := config.LoadConfig(configDir)
	if err != nil {
		return commandRuntime{}, errors.New("load Auth configuration")
	}
	target, err := sanitizedTarget(cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	if err != nil {
		return commandRuntime{}, err
	}
	db, err := database.ConnectDatabase(cfg.Database)
	if err != nil {
		return commandRuntime{}, errors.New("connect configured Auth database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return commandRuntime{}, errors.New("access configured Auth database")
	}
	return commandRuntime{
		target:      target,
		provisioner: benchmark.NewProvisioner(postgres.NewUserRepositoryForManagedSchema(db), security.NewBcryptHasher()),
		close:       func() { _ = sqlDB.Close() },
	}, nil
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var opts options
	flags := flag.NewFlagSet("provision-benchmark-users", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.configDir, "config-dir", "/app/config", "Auth config directory")
	flags.IntVar(&opts.start, "start", 1, "first benchmark account sequence (1..10000)")
	flags.IntVar(&opts.count, "count", 50, "number of benchmark accounts (within 1..10000)")
	flags.BoolVar(&opts.apply, "apply", false, "create missing accounts after confirmation")
	flags.StringVar(&opts.confirm, "confirm", "", "exact confirmation phrase printed by dry-run")
	flags.StringVar(&opts.passwordFile, "password-file", "", "secure file containing one benchmark password")
	flags.BoolVar(&opts.rotatePassword, "rotate-password", false, "atomically rotate an existing canonical benchmark range password")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "explicit non-mutating rotation plan (rotation mode only)")
	flags.BoolVar(&opts.showVersion, "version", false, "print build identity")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, errors.New("unexpected positional arguments")
	}
	if !opts.apply && (opts.confirm != "" || opts.passwordFile != "") {
		return options{}, errors.New("--confirm and --password-file require --apply")
	}
	if opts.dryRun && !opts.rotatePassword {
		return options{}, errors.New("--dry-run is supported only with --rotate-password")
	}
	if opts.rotatePassword && !opts.apply && !opts.dryRun {
		return options{}, errors.New("--rotate-password requires --apply, or explicit --dry-run")
	}
	if opts.rotatePassword && opts.apply && opts.dryRun {
		return options{}, errors.New("--apply and --dry-run cannot be combined")
	}
	if opts.rotatePassword && opts.apply && opts.passwordFile == "" {
		return options{}, errors.New("--rotate-password requires --password-file; interactive input is not allowed")
	}
	if opts.rotatePassword && opts.apply && opts.confirm == "" {
		return options{}, errors.New("--rotate-password requires --confirm")
	}
	return opts, nil
}

func planForMode(provisioner *benchmark.Provisioner, ctx context.Context, opts options) (benchmark.Result, error) {
	if opts.rotatePassword {
		return provisioner.PlanRotation(ctx, opts.start, opts.count)
	}
	return provisioner.Plan(ctx, opts.start, opts.count)
}

func validateApplyConfirmation(opts options, confirmation string, stderr io.Writer) error {
	if opts.confirm == confirmation {
		return nil
	}
	fmt.Fprintf(stderr, "expected confirmation: %s\n", confirmation)
	return errors.New("apply confirmation does not match target and range")
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
	password, err := readPasswordNoEcho(ctx, stdin, reader)
	fmt.Fprintln(stderr)
	if err != nil {
		return nil, err
	}
	fmt.Fprint(stderr, "Confirm benchmark password: ")
	confirmation, err := readPasswordNoEcho(ctx, stdin, reader)
	fmt.Fprintln(stderr)
	if err != nil {
		clear(password)
		return nil, err
	}
	defer clear(confirmation)
	if !bytes.Equal(password, confirmation) {
		clear(password)
		return nil, errors.New("benchmark password confirmation does not match")
	}
	return password, nil
}

func readPasswordNoEcho(ctx context.Context, stdin *os.File, reader *bufio.Reader) ([]byte, error) {
	fd := int(stdin.Fd())
	state, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, errInteractivePasswordTTY
	}
	noEcho := *state
	noEcho.Lflag &^= unix.ECHO
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &noEcho); err != nil {
		return nil, errors.New("disable terminal password echo")
	}
	var restoreOnce sync.Once
	restore := func() { restoreOnce.Do(func() { _ = unix.IoctlSetTermios(fd, unix.TCSETS, state) }) }
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			restore()
			// The CLI owns this standard-input descriptor. Closing it wakes an
			// interrupted terminal read so SIGINT/SIGTERM can exit promptly.
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
	file := os.NewFile(uintptr(fd), "benchmark-password-file")
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, errors.New("read benchmark password file")
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("benchmark password file must be a regular non-symlink file")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("benchmark password file permissions must be owner-only")
	}
	contents, err := io.ReadAll(io.LimitReader(file, maxPasswordBytes+2))
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
	if len(contents) > maxPasswordBytes {
		clear(contents)
		return nil, errors.New("benchmark password file is too large")
	}
	if len(contents) == 0 || bytes.ContainsAny(contents, "\r\n") {
		clear(contents)
		return nil, errors.New("benchmark password file must contain exactly one password with an optional final newline")
	}
	return contents, nil
}

func sanitizedTarget(host string, port int, database string) (string, error) {
	if !safeTargetPart(host, false) || !safeTargetPart(database, true) || port < 1 || port > 65535 {
		return "", errors.New("invalid configured database target")
	}
	return net.JoinHostPort(host, fmt.Sprintf("%d", port)) + "/" + database, nil
}

func confirmationPhrase(identities []benchmark.Identity, target string, rotatePassword bool) string {
	verb := "CREATE"
	if rotatePassword {
		verb = "ROTATE PASSWORD"
	}
	return fmt.Sprintf("%s %s..%s ON %s", verb, identities[0].Username, identities[len(identities)-1].Username, target)
}

func safeTargetPart(value string, rejectSlash bool) bool {
	if value == "" || value != strings.TrimSpace(value) || (rejectSlash && strings.Contains(value, "/")) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func printResult(w io.Writer, apply, rotatePassword bool, target string, identities []benchmark.Identity, confirmation string, result benchmark.Result, elapsed time.Duration) {
	mode := "dry-run"
	if rotatePassword {
		mode = "rotate-password dry-run"
	}
	if apply {
		mode = "apply"
		if rotatePassword {
			mode = "rotate-password"
		}
	}
	fmt.Fprintf(w, "Mode: %s\nTarget: %s\nRange: %s..%s\n\n", mode, target, identities[0].Username, identities[len(identities)-1].Username)
	if apply && rotatePassword {
		fmt.Fprintf(w, "Rotated: %d\nConflicts: %d\n", result.Rotated, result.Conflicts)
	} else if apply {
		fmt.Fprintf(w, "Created: %d\nSkipped: %d\nConflicts: %d\n", result.Created, result.Skipped, result.Conflicts)
	} else if rotatePassword {
		wouldRotate := 0
		for _, entry := range result.Entries {
			if entry.Status == benchmark.StatusWouldRotate {
				wouldRotate++
			}
		}
		fmt.Fprintf(w, "Would rotate: %d\nConflicts: %d\nNo changes applied.\n", wouldRotate, result.Conflicts)
		fmt.Fprintf(w, "Apply confirmation: %s\n", confirmation)
	} else {
		wouldCreate := 0
		for _, entry := range result.Entries {
			if entry.Status == benchmark.StatusWouldCreate {
				wouldCreate++
			}
		}
		fmt.Fprintf(w, "Would create: %d\nExisting canonical identities: %d\nConflicts: %d\n\nExisting password verification: not performed in dry-run\nNo changes applied.\n", wouldCreate, result.Existing, result.Conflicts)
		fmt.Fprintf(w, "Apply confirmation: %s\n", confirmation)
	}
	if result.StoppedAt != nil {
		fmt.Fprintf(w, "Stopped at: %s\n", result.StoppedAt.Username)
	}
	fmt.Fprintf(w, "Elapsed: %s\n", elapsed.Round(time.Millisecond))
}

func safeError(err error) string {
	switch {
	case errors.Is(err, benchmark.ErrInvalidRange):
		return "start/count must select benchmark accounts 001 through 10000"
	case errors.Is(err, benchmark.ErrConflicts):
		return "one or more benchmark identities conflict; no new users were created"
	case errors.Is(err, benchmark.ErrApplyStopped):
		return "provisioning stopped after a safe verification failure"
	case errors.Is(err, benchmark.ErrRotateStopped):
		return "password rotation stopped safely; no password changes were committed"
	case errors.Is(err, context.Canceled):
		return "provisioning cancelled"
	case errors.Is(err, errInteractivePasswordTTY):
		return errInteractivePasswordTTY.Error()
	default:
		// Config and database libraries can include a DSN or path in their
		// errors. Keep command output deliberately high-level.
		return "safe provisioning error; review the configuration and command inputs"
	}
}

func progressReporter(w io.Writer) func(completed, total int) {
	return func(completed, total int) {
		if completed == total || completed%500 == 0 {
			fmt.Fprintf(w, "Progress: %d/%d\n", completed, total)
		}
	}
}

func clear(value []byte) {
	for i := range value {
		value[i] = 0
	}
}

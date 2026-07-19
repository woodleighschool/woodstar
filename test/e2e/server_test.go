package e2e

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/test/e2e/adminapi"
)

const (
	testBinaryEnvironment             = "WOODSTAR_TEST_BINARY"
	testDatabaseEnvironment           = "WOODSTAR_TEST_DATABASE_URL"
	readinessTimeout                  = 20 * time.Second
	testClientTimeout                 = 30 * time.Second
	processShutdownTimeout            = 20 * time.Second
	databaseOperationTimeout          = 10 * time.Second
	serverLogTailLimit                = 64 << 10
	testStorageCapabilityKeyByteCount = 32
	testStorageTransferTTL            = 7 * time.Minute
)

type testServer struct {
	BaseURL              string
	Client               *http.Client
	Admin                *adminapi.ClientWithResponses
	AdminHTTP            *http.Client
	DatabaseURL          string
	StorageRoot          string
	StorageCapabilityKey string
	CACertificate        []byte
	CACertificatePath    string

	logPath    string
	redactions []string
}

type testTLS struct {
	certificatePath string
	privateKeyPath  string
	caCertificate   []byte
	caPath          string
}

type serverProcess struct {
	command *exec.Cmd
	done    chan struct{}
	waitErr error
}

type binaryCache struct {
	once      sync.Once
	path      string
	buildRoot string
	err       error
}

var compiledTestBinary binaryCache

func TestMain(m *testing.M) {
	exitCode := m.Run()
	if err := compiledTestBinary.cleanup(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "remove compiled Woodstar test binary: %v\n", err)
		if exitCode == 0 {
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

func startTestServer(t *testing.T) *testServer {
	t.Helper()

	baseDatabaseURL := strings.TrimSpace(os.Getenv(testDatabaseEnvironment))
	if baseDatabaseURL == "" {
		t.Fatalf("%s is required for end-to-end server tests", testDatabaseEnvironment)
	}

	binaryPath := testBinary(t)
	root := t.TempDir()
	databaseURL := createTestDatabase(t, baseDatabaseURL)
	tlsMaterial := createTestTLS(t, root)
	storageRoot := filepath.Join(root, "storage")
	if err := os.Mkdir(storageRoot, 0o700); err != nil {
		t.Fatalf("create test storage root: %v", err)
	}

	port := allocatePort(t)
	baseURL := "https://localhost:" + strconv.Itoa(port)
	client := verifyingClient(t, tlsMaterial.caCertificate)
	storageCapabilityKey := randomHex(t, testStorageCapabilityKeyByteCount)
	logPath := filepath.Join(root, "woodstar.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("create server log: %v", err)
	}
	t.Cleanup(func() { _ = logFile.Close() })

	server := &testServer{
		BaseURL:              baseURL,
		Client:               client,
		DatabaseURL:          databaseURL,
		StorageRoot:          storageRoot,
		StorageCapabilityKey: storageCapabilityKey,
		CACertificate:        append([]byte(nil), tlsMaterial.caCertificate...),
		CACertificatePath:    tlsMaterial.caPath,
		logPath:              logPath,
	}
	server.redact(storageCapabilityKey, databaseURL)
	if parsedDatabaseURL, parseErr := url.Parse(databaseURL); parseErr == nil && parsedDatabaseURL.User != nil {
		if password, ok := parsedDatabaseURL.User.Password(); ok {
			server.redact(password)
		}
	}

	command := woodstarCommand(
		binaryPath,
		port,
		baseURL,
		databaseURL,
		storageRoot,
		storageCapabilityKey,
		tlsMaterial,
		logFile,
	)
	process, err := startProcess(command)
	if err != nil {
		t.Fatalf("start Woodstar: %v\n%s", err, server.logs())
	}
	t.Cleanup(func() {
		stopProcess(t, "Woodstar", process)
		if t.Failed() {
			t.Logf("Woodstar server logs (tail):\n%s", server.logs())
		}
	})

	if err := waitForHealth(t.Context(), client, baseURL, process); err != nil {
		t.Fatalf("wait for Woodstar readiness: %v", err)
	}

	return server
}

func (server *testServer) redact(values ...string) {
	server.redactions = append(server.redactions, values...)
}

func (server *testServer) logs() string {
	return safeLogTail(server.logPath, server.redactions)
}

func testBinary(t *testing.T) string {
	t.Helper()

	compiledTestBinary.once.Do(compiledTestBinary.resolve)
	if compiledTestBinary.err != nil {
		t.Fatalf("prepare Woodstar test binary: %v", compiledTestBinary.err)
	}
	return compiledTestBinary.path
}

func (cache *binaryCache) resolve() {
	configuredPath := strings.TrimSpace(os.Getenv(testBinaryEnvironment))
	if configuredPath != "" {
		cache.path, cache.err = existingExecutable(configuredPath)
		return
	}

	repositoryRoot, err := findRepositoryRoot()
	if err != nil {
		cache.err = err
		return
	}
	cache.buildRoot, err = os.MkdirTemp("", "woodstar-e2e-binary-")
	if err != nil {
		cache.err = fmt.Errorf("create binary build directory: %w", err)
		return
	}
	cache.path = filepath.Join(cache.buildRoot, "woodstar")
	command := exec.Command("go", "build", "-o", cache.path, "./cmd/woodstar")
	command.Dir = repositoryRoot
	output, err := command.CombinedOutput()
	if err != nil {
		cache.err = fmt.Errorf("build ./cmd/woodstar: %w\n%s", err, tail(output, serverLogTailLimit))
	}
}

func (cache *binaryCache) cleanup() error {
	if cache.buildRoot == "" {
		return nil
	}
	return os.RemoveAll(cache.buildRoot)
}

func existingExecutable(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", testBinaryEnvironment, err)
	}
	info, err := os.Stat(absolutePath)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", testBinaryEnvironment, err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("%s must name an executable file", testBinaryEnvironment)
	}
	return absolutePath, nil
}

func findRepositoryRoot() (string, error) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("locate integration test source")
	}
	repositoryRoot := filepath.Dir(filepath.Dir(filepath.Dir(sourceFile)))
	if _, err := os.Stat(filepath.Join(repositoryRoot, "go.mod")); err != nil {
		return "", fmt.Errorf("locate repository go.mod: %w", err)
	}
	return repositoryRoot, nil
}

func createTestDatabase(t *testing.T, baseURL string) string {
	t.Helper()

	databaseName := "woodstar_integration_" + randomHex(t, 8)
	adminURL, databaseURL, err := databaseURLs(baseURL, databaseName)
	if err != nil {
		t.Fatalf("parse %s: %v", testDatabaseEnvironment, err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), databaseOperationTimeout)
	defer cancel()
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect to PostgreSQL test server: %v", err)
	}
	defer func() { _ = admin.Close(ctx) }()

	identifier := pgx.Identifier{databaseName}.Sanitize()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), databaseOperationTimeout)
		defer cleanupCancel()
		cleanupAdmin, cleanupErr := pgx.Connect(cleanupCtx, adminURL)
		if cleanupErr != nil {
			t.Errorf("connect to drop test database %s: %v", databaseName, cleanupErr)
			return
		}
		defer func() { _ = cleanupAdmin.Close(cleanupCtx) }()
		if _, cleanupErr = cleanupAdmin.Exec(
			cleanupCtx,
			"DROP DATABASE IF EXISTS "+identifier+" WITH (FORCE)",
		); cleanupErr != nil {
			t.Errorf("drop test database %s: %v", databaseName, cleanupErr)
		}
	})

	if _, err := admin.Exec(ctx, "CREATE DATABASE "+identifier); err != nil {
		t.Fatalf("create test database %s: %v", databaseName, err)
	}
	return databaseURL
}

func databaseURLs(baseURL string, databaseName string) (string, string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", "", err
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return "", "", fmt.Errorf("unsupported PostgreSQL URL scheme %q", parsed.Scheme)
	}
	admin := *parsed
	admin.Path = "/postgres"
	admin.RawPath = ""
	target := *parsed
	target.Path = "/" + databaseName
	target.RawPath = ""
	return admin.String(), target.String(), nil
}

func createTestTLS(t *testing.T, root string) testTLS {
	t.Helper()

	now := time.Now()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate test CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          randomSerial(t),
		Subject:               pkix.Name{CommonName: "Woodstar integration test CA"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create test CA certificate: %v", err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate test server key: %v", err)
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: randomSerial(t),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost", "host.testcontainers.internal"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	serverDER, err := x509.CreateCertificate(
		rand.Reader,
		serverTemplate,
		caTemplate,
		&serverKey.PublicKey,
		caKey,
	)
	if err != nil {
		t.Fatalf("create test server certificate: %v", err)
	}
	serverKeyDER, err := x509.MarshalPKCS8PrivateKey(serverKey)
	if err != nil {
		t.Fatalf("marshal test server key: %v", err)
	}

	caPath := filepath.Join(root, "ca.pem")
	certificatePath := filepath.Join(root, "server.pem")
	privateKeyPath := filepath.Join(root, "server-key.pem")
	// The CA is public and must be readable by a later bind-mounted Orbit container.
	if err := os.WriteFile(caPath, caPEM, 0o644); err != nil {
		t.Fatalf("write test CA certificate: %v", err)
	}
	if err := os.WriteFile(
		certificatePath,
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER}),
		0o600,
	); err != nil {
		t.Fatalf("write test server certificate: %v", err)
	}
	if err := os.WriteFile(
		privateKeyPath,
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: serverKeyDER}),
		0o600,
	); err != nil {
		t.Fatalf("write test server key: %v", err)
	}

	return testTLS{
		certificatePath: certificatePath,
		privateKeyPath:  privateKeyPath,
		caCertificate:   caPEM,
		caPath:          caPath,
	}
}

func randomSerial(t *testing.T) *big.Int {
	t.Helper()

	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, limit)
	if err != nil {
		t.Fatalf("generate certificate serial number: %v", err)
	}
	return serial
}

func randomHex(t *testing.T, byteCount int) string {
	t.Helper()

	value := make([]byte, byteCount)
	if _, err := rand.Read(value); err != nil {
		t.Fatalf("generate random test value: %v", err)
	}
	return hex.EncodeToString(value)
}

func allocatePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate test server port: %v", err)
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		t.Fatalf("test listener address has type %T", listener.Addr())
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("release allocated test server port: %v", err)
	}
	return address.Port
}

func verifyingClient(t *testing.T, caCertificate []byte) *http.Client {
	t.Helper()

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caCertificate) {
		t.Fatal("load test CA certificate")
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create test cookie jar: %v", err)
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    roots,
		},
	}
	t.Cleanup(transport.CloseIdleConnections)
	return &http.Client{Transport: transport, Jar: jar, Timeout: testClientTimeout}
}

func woodstarCommand(
	binaryPath string,
	port int,
	baseURL string,
	databaseURL string,
	storageRoot string,
	storageCapabilityKey string,
	tlsMaterial testTLS,
	logFile *os.File,
) *exec.Cmd {
	command := exec.Command(
		binaryPath,
		"serve",
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(port),
		"--url", baseURL,
		"--tls-cert-file", tlsMaterial.certificatePath,
		"--tls-key-file", tlsMaterial.privateKeyPath,
		"--log-level", "info",
	)
	command.Env = append(
		withoutWoodstarEnvironment(os.Environ()),
		"WOODSTAR_DATABASE_URL="+databaseURL,
		"WOODSTAR_STORAGE_CAPABILITY_KEY="+storageCapabilityKey,
		"WOODSTAR_STORAGE_KIND=file",
		"WOODSTAR_STORAGE_FILE_ROOT="+storageRoot,
		"WOODSTAR_STORAGE_TRANSFER_TTL="+testStorageTransferTTL.String(),
	)
	command.Stdout = logFile
	command.Stderr = logFile
	return command
}

func withoutWoodstarEnvironment(environment []string) []string {
	cleaned := make([]string, 0, len(environment))
	for _, entry := range environment {
		if !strings.HasPrefix(entry, "WOODSTAR_") {
			cleaned = append(cleaned, entry)
		}
	}
	return cleaned
}

func startProcess(command *exec.Cmd) (*serverProcess, error) {
	if err := command.Start(); err != nil {
		return nil, err
	}
	process := &serverProcess{command: command, done: make(chan struct{})}
	go func() {
		process.waitErr = command.Wait()
		close(process.done)
	}()
	return process, nil
}

func waitForHealth(
	parent context.Context,
	client *http.Client,
	baseURL string,
	process *serverProcess,
) error {
	ctx, cancel := context.WithTimeout(parent, readinessTimeout)
	defer cancel()

	backoff := 10 * time.Millisecond
	var lastErr error
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/healthz", nil)
		if err != nil {
			return fmt.Errorf("create health request: %w", err)
		}
		response, err := client.Do(request)
		if err == nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("health endpoint returned %s", response.Status)
		} else {
			lastErr = err
		}

		timer := time.NewTimer(backoff)
		select {
		case <-process.done:
			timer.Stop()
			if process.waitErr == nil {
				return errors.New("Woodstar exited before readiness")
			}
			return fmt.Errorf("Woodstar exited before readiness: %w", process.waitErr)
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("health check deadline: %w (last error: %w)", ctx.Err(), lastErr)
		case <-timer.C:
		}
		backoff = min(backoff*2, 250*time.Millisecond)
	}
}

func stopProcess(t *testing.T, name string, process *serverProcess) {
	t.Helper()

	select {
	case <-process.done:
		t.Errorf("%s exited unexpectedly: %v", name, process.waitErr)
		return
	default:
	}

	if err := process.command.Process.Signal(os.Interrupt); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			<-process.done
			t.Errorf("%s exited unexpectedly: %v", name, process.waitErr)
			return
		}
		t.Errorf("signal %s process: %v", name, err)
	}

	timer := time.NewTimer(processShutdownTimeout)
	defer timer.Stop()
	select {
	case <-process.done:
		if process.waitErr != nil {
			t.Errorf("wait for %s process: %v", name, process.waitErr)
		}
	case <-timer.C:
		if err := process.command.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			t.Errorf("kill %s process after shutdown timeout: %v", name, err)
		}
		<-process.done
		t.Errorf("%s did not stop within %s", name, processShutdownTimeout)
	}
}

func safeLogTail(path string, redactions []string) string {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "read server log: " + err.Error()
	}
	logs := string(contents)
	for _, secret := range redactions {
		if secret != "" {
			logs = strings.ReplaceAll(logs, secret, "[REDACTED]")
		}
	}
	logs = tail([]byte(logs), serverLogTailLimit)
	if logs == "" {
		return "(no server output)"
	}
	return logs
}

func tail(contents []byte, limit int) string {
	if len(contents) > limit {
		contents = contents[len(contents)-limit:]
	}
	return string(contents)
}

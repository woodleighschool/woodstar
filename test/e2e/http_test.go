package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/woodleighschool/woodstar/test/e2e/adminapi"
)

type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func (transport bearerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Header = request.Header.Clone()
	request.Header.Set("Authorization", "Bearer "+transport.token)
	return transport.base.RoundTrip(request)
}

func provisionAdmin(
	t *testing.T,
	server *testServer,
	email string,
	name string,
	password string,
) {
	t.Helper()
	initialPassword := password + "-initial"
	server.redact(password, initialPassword)
	runUserCommand(
		t,
		server,
		"create",
		"--email", email,
		"--name", name,
		"--role", "viewer",
		"--password", initialPassword,
	)
	runUserCommand(t, server, "set-role", "--email", email, "--role", "admin")
	runUserCommand(t, server, "set-password", "--email", email, "--password", password)

	sessionClient := newAdminAPIClient(t, server.BaseURL, server.Client)
	login, err := sessionClient.CreateSessionWithResponse(t.Context(), adminapi.CreateSessionJSONRequestBody{
		Email:    openapi_types.Email(email),
		Password: password,
	})
	login = requireAPIResponse(t, "log in as persisted administrator", http.StatusOK, login, err)
	if login.JSON200 == nil || login.JSON200.Id <= 0 || string(login.JSON200.Email) != email {
		t.Fatalf("persisted user = %+v, want %q", login.JSON200, email)
	}

	secureSession := false
	for _, cookie := range login.HTTPResponse.Cookies() {
		secureSession = secureSession || cookie.Name == "woodstar_session" && cookie.Secure
	}
	if !secureSession {
		t.Fatal("login response did not issue a secure session cookie")
	}

	rotated, err := sessionClient.RotateAccountApiKeyWithResponse(t.Context())
	rotated = requireAPIResponse(t, "rotate administrator API key", http.StatusCreated, rotated, err)
	if rotated.JSON201 == nil || rotated.JSON201.ApiKey == nil || *rotated.JSON201.ApiKey == "" {
		t.Fatalf("rotated account = %+v, want API key", rotated.JSON201)
	}

	apiKey := *rotated.JSON201.ApiKey
	server.redact(apiKey)
	server.AdminHTTP = verifyingClient(t, server.CACertificate)
	server.AdminHTTP.Transport = bearerTransport{
		base:  server.AdminHTTP.Transport,
		token: apiKey,
	}
	server.Admin = newAdminAPIClient(t, server.BaseURL, server.AdminHTTP)

	account, err := server.Admin.GetAccountWithResponse(t.Context())
	account = requireAPIResponse(t, "get account with administrator API key", http.StatusOK, account, err)
	if account.JSON200 == nil || account.JSON200.User.Id != login.JSON200.Id {
		t.Fatalf("API-key account = %+v, want persisted user %d", account.JSON200, login.JSON200.Id)
	}
}

func runUserCommand(t *testing.T, server *testServer, args ...string) {
	t.Helper()
	commandArgs := []string{"user", "--database-url", server.DatabaseURL}
	commandArgs = append(commandArgs, args...)
	command := exec.CommandContext(t.Context(), server.BinaryPath, commandArgs...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run Woodstar user command: %v\n%s", err, output)
	}
}

func createAgentSecret(
	t *testing.T,
	server *testServer,
	agent adminapi.AgentSecretCreateAgent,
	value string,
) *adminapi.AgentSecret {
	t.Helper()

	response, err := server.Admin.CreateAgentSecretWithResponse(
		t.Context(),
		adminapi.CreateAgentSecretJSONRequestBody{
			Agent: agent,
			Value: value,
		},
	)
	response = requireAPIResponse(t, "create agent secret", http.StatusCreated, response, err)
	if response.JSON201 == nil {
		t.Fatal("create agent secret returned no JSON body")
	}
	return response.JSON201
}

func directPackageInstallerUpload(
	t *testing.T,
	target *adminapi.MunkiPackageInstallerUploadTarget,
) adminapi.MunkiDirectUploadAction {
	t.Helper()

	if target == nil {
		t.Fatal("create package installer upload target returned no JSON body")
	}
	upload, err := target.Upload.AsMunkiDirectUploadAction()
	if err != nil {
		t.Fatalf("decode direct package installer upload action: %v", err)
	}
	return upload
}

func newAdminAPIClient(
	t *testing.T,
	baseURL string,
	httpClient *http.Client,
) *adminapi.ClientWithResponses {
	t.Helper()

	client, err := adminapi.NewClientWithResponses(baseURL, adminapi.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("create generated admin API client: %v", err)
	}
	return client
}

func requireAPIResponse[T interface {
	comparable
	GetBody() []byte
	StatusCode() int
}](t *testing.T, operation string, wantStatus int, response T, err error) T {
	t.Helper()

	var zero T
	if err != nil {
		t.Fatalf("%s: %v", operation, err)
	}
	if response == zero {
		t.Fatalf("%s returned no response", operation)
	}
	if response.StatusCode() != wantStatus {
		t.Fatalf(
			"%s status = %d, want %d: %s",
			operation,
			response.StatusCode(),
			wantStatus,
			response.GetBody(),
		)
	}
	return response
}

func postJSON(
	t *testing.T,
	client *http.Client,
	requestURL string,
	body any,
	target any,
) *http.Response {
	t.Helper()

	var requestBody io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode POST %s JSON request: %v", requestURL, err)
		}
		requestBody = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, requestURL, requestBody)
	if err != nil {
		t.Fatalf("create POST %s request: %v", requestURL, err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send POST %s request: %v", requestURL, err)
	}
	responseBody := readAndClose(t, response)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("POST %s status = %d, want %d", requestURL, response.StatusCode, http.StatusOK)
	}
	if target != nil {
		if err := json.Unmarshal(responseBody, target); err != nil {
			t.Fatalf("decode POST %s JSON response: %v", requestURL, err)
		}
	}
	return response
}

func readAndClose(t *testing.T, response *http.Response) []byte {
	t.Helper()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		_ = response.Body.Close()
		t.Fatalf("read HTTP response: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close HTTP response: %v", err)
	}
	return body
}

func drainAndClose(t *testing.T, response *http.Response) {
	t.Helper()

	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		_ = response.Body.Close()
		t.Fatalf("drain HTTP response: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close HTTP response: %v", err)
	}
}

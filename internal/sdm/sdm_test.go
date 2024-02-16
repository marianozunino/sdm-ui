package sdm

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	_ "os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSdmBehavior = "TEST_SDM_BEHAVIOR"

type testBehavior int

func (b testBehavior) String() string {
	return fmt.Sprintf("%d", b)
}

const (
	cmdReadySuccessBehavior testBehavior = iota
	cmdReadyNoAccountBehavior
	cmdReadyErrorBehavior
	cmdLoginSuccessBehavior
	cmdLoginErrorNoAccountBehavior
	cmdLoginErrorUnknownBehavior
	cmdLoginInvalidCredentialsBehavior
	cmdLogoutSuccessBehavior
	cmdLogoutNotAuthenticatedBehavior
	cmdLogoutErrorBehavior

	cmdStatusSuccessBehavior
	cmdStatusNotAuthenticatedBehavior
	cmdStatusErrorBehavior

	cmdConnectSuccessBehavior
	cmdConnectNotAuthenticatedBehavior
	cmdConnectResourceNotFoundBehavior
	cmdConnectErrorBehavior
)

func strPtr(s string) *string {
	return &s
}

// Helper functions for command execution
func executeCommand(want []string, output string, exitCode int) {
	if args := os.Args[1:]; !slices.Equal(want, args) {
		fmt.Printf("expected arguments %q, got %q\n", want, args)
		os.Exit(1)
	}
	fmt.Println(output)
	os.Exit(exitCode)
}

func TestMain(m *testing.M) {
	behavior := os.Getenv(testSdmBehavior)
	switch behavior {
	case "":
		os.Exit(m.Run())

	case cmdReadySuccessBehavior.String():
		executeCommand([]string{"ready"}, `{"account":"some.account@email.com","listener_running":true,"state_loaded":true,"is_linked":true}`, 0)
	case cmdReadyNoAccountBehavior.String():
		executeCommand([]string{"ready"}, `{"listener_running":true,"state_loaded":true,"is_linked":true}`, 0)
	case cmdReadyErrorBehavior.String():
		executeCommand([]string{"ready"}, ``, 1)

	case cmdLoginSuccessBehavior.String():
		executeCommand([]string{"login", "--email", "some.account@email.com"}, `logged in`, 0)
	case cmdLoginErrorNoAccountBehavior.String():
		executeCommand([]string{"login", "--email", "some.account@email.com"}, `This email doesn't have a strongDM account.`, 1)
	case cmdLoginErrorUnknownBehavior.String():
		executeCommand([]string{"login", "--email", "some.account@email.com"}, `cannot ask for password`, 1)
	case cmdLoginInvalidCredentialsBehavior.String():
		executeCommand([]string{"login", "--email", "some.account@email.com"}, `access denied\n`, 1)

	case cmdLogoutSuccessBehavior.String():
		executeCommand([]string{"logout"}, `logged out`, 0)
	case cmdLogoutNotAuthenticatedBehavior.String():
		executeCommand([]string{"logout"}, `You are not authenticated. Please login again.`, 9)
	case cmdLogoutErrorBehavior.String():
		executeCommand([]string{"logout"}, ``, 1)

	case cmdStatusSuccessBehavior.String():
		executeCommand([]string{"status"}, `random output`, 0)
	case cmdStatusNotAuthenticatedBehavior.String():
		executeCommand([]string{"status"}, `You are not authenticated. Please login again.`, 9)
	case cmdStatusErrorBehavior.String():
		executeCommand([]string{"status"}, ``, 1)

	case cmdConnectSuccessBehavior.String():
		executeCommand([]string{"connect", "resource_name"}, `random output`, 0)
	case cmdConnectErrorBehavior.String():
		executeCommand([]string{"connect", "resource_name"}, ``, 1)
	case cmdConnectNotAuthenticatedBehavior.String():
		executeCommand([]string{"connect", "resource_name"}, `You are not authenticated. Please login again.`, 9)
	case cmdConnectResourceNotFoundBehavior.String():
		executeCommand([]string{"connect", "resource_name"}, `Cannot find datasource named ''`, 1)

	default:
		log.Fatalf("unknown behavior %q", behavior)
	}
}

func TestSDMClient_Ready(t *testing.T) {
	tests := []struct {
		name        string
		behavior    string
		expected    SdmReady
		expectedErr bool
		panics      bool
	}{
		{
			name:     "SuccessfulReady",
			behavior: cmdReadySuccessBehavior.String(),
			expected: SdmReady{
				IsLinked:        true,
				StateLoaded:     true,
				ListenerRunning: true,
				Account:         strPtr("some.account@email.com"),
			},
			expectedErr: false,
		},
		{
			name:     "NoAccount",
			behavior: cmdReadyNoAccountBehavior.String(),
			expected: SdmReady{
				IsLinked:        true,
				StateLoaded:     true,
				ListenerRunning: true,
				Account:         nil,
			},
			expectedErr: false,
		},
		{
			name:     "Error",
			behavior: cmdReadyErrorBehavior.String(),
			panics:   true,
			expected: SdmReady{
				IsLinked:        true,
				StateLoaded:     true,
				ListenerRunning: true,
				Account:         nil,
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testExe, err := os.Executable()
			require.NoError(t, err, "can't determine current executable")

			sdmWrapper := SDMClient{Exe: testExe}
			os.Setenv(testSdmBehavior, tt.behavior)
			defer os.Unsetenv(testSdmBehavior)

			if tt.panics {
				require.Panics(t, func() {
					sdmWrapper.Ready()
				}, "Test %s should panic", tt.name)
				return
			}

			got, err := sdmWrapper.Ready()

			if tt.expectedErr {
				require.Error(t, err, "Test %s should return an error", tt.name)
				return
			}

			require.NoError(t, err, "Test %s failed", tt.name)
			assert.Equal(t, tt.expected, got, "Test %s failed: unexpected result", tt.name)
		})
	}
}

func TestSDMClient_Login(t *testing.T) {
	tests := []struct {
		name           string
		behavior       string
		expected       error
		email          string
		password       string
		expectedErr    bool
		expectedErrMsg string
		panics         bool
	}{
		{
			name:        "SuccessfulLogin",
			behavior:    cmdLoginSuccessBehavior.String(),
			email:       "some.account@email.com",
			expectedErr: false,
		},
		{
			name:           "ErrorNoAccount",
			behavior:       cmdLoginErrorNoAccountBehavior.String(),
			email:          "some.account@email.com",
			expectedErr:    true,
			expectedErrMsg: `This email doesn't have a strongDM account.`,
		},
		{
			name:           "ErrorUnknown",
			behavior:       cmdLoginErrorUnknownBehavior.String(),
			email:          "some.account@email.com",
			expectedErr:    true,
			expectedErrMsg: `cannot ask for password`,
		},
		{
			name:           "ErrorInvalidCredentials",
			behavior:       cmdLoginInvalidCredentialsBehavior.String(),
			email:          "some.account@email.com",
			expectedErr:    true,
			expectedErrMsg: `access denied\n`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testExe, err := os.Executable()
			require.NoError(t, err, "can't determine current executable")

			sdmWrapper := SDMClient{Exe: testExe}
			os.Setenv(testSdmBehavior, tt.behavior)
			defer os.Unsetenv(testSdmBehavior)

			if tt.panics {
				require.Panics(t, func() {
					sdmWrapper.Login(tt.email, tt.password)
				}, "Test %s should panic", tt.name)
				return
			}

			got := sdmWrapper.Login(tt.email, tt.password)

			if tt.expectedErr {
				require.Error(t, got, "Test %s should return an error", tt.name)
				assert.Contains(t, got.Error(), tt.expectedErrMsg, "Test %s failed: unexpected error message", tt.name)
				return
			}

			require.NoError(t, got, "Test %s failed", tt.name)
		})
	}
}

func TestSDMClient_Logout(t *testing.T) {
	tests := []struct {
		name            string
		behavior        testBehavior
		expected        error
		expectedErr     bool
		expectedErrMsg  string
		panics          bool
		expectedErrCode SDMErrorCode
	}{
		{
			name:        "SuccessfulLogout",
			behavior:    cmdLogoutSuccessBehavior,
			expectedErr: false,
		},
		{
			name:            "ErrorNotAuthenticated",
			behavior:        cmdLogoutNotAuthenticatedBehavior,
			expectedErr:     true,
			expectedErrCode: Unauthorized,
		},
		{
			name:            "ErrorUnknown",
			behavior:        cmdLogoutErrorBehavior,
			expectedErr:     true,
			expectedErrCode: Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testExe, err := os.Executable()
			require.NoError(t, err, "can't determine current executable")

			sdmWrapper := SDMClient{Exe: testExe}
			os.Setenv(testSdmBehavior, tt.behavior.String())
			defer os.Unsetenv(testSdmBehavior)

			if tt.panics {
				require.Panics(t, func() {
					sdmWrapper.Logout()
				}, "Test %s should panic", tt.name)
				return
			}

			got := sdmWrapper.Logout()

			if tt.expectedErr {
				require.Error(t, got, "Test %s should return an error", tt.name)
				assert.Contains(t, got.Error(), tt.expectedErrMsg, "Test %s failed: unexpected error message", tt.name)

				if tt.expectedErrCode != 0 {
					assert.Equal(t, tt.expectedErrCode, got.(SDMError).Code, "Test %s failed: unexpected error code", tt.name)
				}

				return
			}

			require.NoError(t, got, "Test %s failed", tt.name)
		})
	}
}

func TestSDMClient_Status(t *testing.T) {
	tests := []struct {
		name           string
		behavior       testBehavior
		expected       error
		expectedErr    bool
		expectedErrMsg string
		panics         bool
		writer         io.Writer
	}{
		{
			name:        "SuccessfulStatus",
			behavior:    cmdStatusSuccessBehavior,
			expectedErr: false,
			writer:      bytes.NewBuffer(nil),
		},
		{
			name:        "ErrorUnknown",
			behavior:    cmdStatusErrorBehavior,
			expectedErr: true,
			writer:      bytes.NewBuffer(nil),
		},
		{
			name:        "NotAuthenticated",
			behavior:    cmdStatusNotAuthenticatedBehavior,
			expectedErr: true,
			writer:      bytes.NewBuffer(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testExe, err := os.Executable()
			require.NoError(t, err, "can't determine current executable")

			sdmWrapper := SDMClient{Exe: testExe}
			os.Setenv(testSdmBehavior, tt.behavior.String())
			defer os.Unsetenv(testSdmBehavior)

			if tt.panics {
				require.Panics(t, func() {
					sdmWrapper.Status(tt.writer)
				}, "Test %s should panic", tt.name)
				return
			}

			got := sdmWrapper.Status(tt.writer)

			if tt.expectedErr {
				require.Error(t, got, "Test %s should return an error", tt.name)
				assert.Contains(t, got.Error(), tt.expectedErrMsg, "Test %s failed: unexpected error message", tt.name)
				return
			}

			require.NoError(t, got, "Test %s failed", tt.name)

			assert.Contains(t, tt.writer.(*bytes.Buffer).String(), "random output", "Test %s failed: unexpected output", tt.name)
		})
	}
}

func TestSDMClient_Connect(t *testing.T) {
	tests := []struct {
		name            string
		behavior        testBehavior
		expected        error
		expectedErr     bool
		expectedErrCode SDMErrorCode
		panics          bool
		resource        string
	}{
		{
			name:        "SuccessfulConnect",
			behavior:    cmdConnectSuccessBehavior,
			expectedErr: false,
			resource:    "resource_name",
		},
		{
			name:            "ErrorUnknown",
			behavior:        cmdConnectErrorBehavior,
			expectedErr:     true,
			expectedErrCode: Unknown,
			resource:        "resource_name",
		},
		{
			name:            "NotAuthenticated",
			behavior:        cmdConnectNotAuthenticatedBehavior,
			expectedErr:     true,
			expectedErrCode: Unauthorized,
			resource:        "resource_name",
		},
		{
			name:            "ResourceNameMissing",
			behavior:        cmdConnectResourceNotFoundBehavior,
			expectedErr:     true,
			expectedErrCode: ResourceNotFound,
			resource:        "resource_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testExe, err := os.Executable()
			require.NoError(t, err, "can't determine current executable")

			sdmWrapper := SDMClient{Exe: testExe}
			os.Setenv(testSdmBehavior, tt.behavior.String())
			defer os.Unsetenv(testSdmBehavior)

			if tt.panics {
				require.Panics(t, func() {
					sdmWrapper.Connect(tt.resource)
				}, "Test %s should panic", tt.name)
				return
			}

			got := sdmWrapper.Connect(tt.resource)

			if tt.expectedErr {
				require.Error(t, got, "Test %s should return an error", tt.name)

				expectedErr, ok := got.(SDMError)
				require.True(t, ok, "Test %s failed: unexpected error type", tt.name)
				assert.Equal(t, tt.expectedErrCode, expectedErr.Code, "Test %s failed: unexpected error code", tt.name)

				return
			}

			require.NoError(t, got, "Test %s failed", tt.name)

		})
	}
}

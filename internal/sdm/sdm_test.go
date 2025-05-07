package sdm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSdmBehavior = "TEST_SDM_BEHAVIOR"

// Test behaviors
type TestBehavior int

const (
	cmdReadySuccessBehavior TestBehavior = iota
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

// String conversion for TestBehavior
func (tb TestBehavior) String() string {
	behaviors := []string{
		"cmdReadySuccessBehavior",
		"cmdReadyNoAccountBehavior",
		"cmdReadyErrorBehavior",
		"cmdLoginSuccessBehavior",
		"cmdLoginErrorNoAccountBehavior",
		"cmdLoginErrorUnknownBehavior",
		"cmdLoginInvalidCredentialsBehavior",
		"cmdLogoutSuccessBehavior",
		"cmdLogoutNotAuthenticatedBehavior",
		"cmdLogoutErrorBehavior",
		"cmdStatusSuccessBehavior",
		"cmdStatusNotAuthenticatedBehavior",
		"cmdStatusErrorBehavior",
		"cmdConnectSuccessBehavior",
		"cmdConnectNotAuthenticatedBehavior",
		"cmdConnectResourceNotFoundBehavior",
		"cmdConnectErrorBehavior",
	}

	if int(tb) < 0 || int(tb) >= len(behaviors) {
		return fmt.Sprintf("TestBehavior(%d)", tb)
	}
	return behaviors[tb]
}

// TestMain handles special behavior when running as a subprocess
func TestMain(m *testing.M) {
	behavior := os.Getenv(testSdmBehavior)

	// Execution as a normal test
	if behavior == "" {
		os.Exit(m.Run())
	}

	// Map behavior to command output and exit code
	outputMap := map[string]struct {
		output   string
		exitCode int
	}{
		cmdReadySuccessBehavior.String():            {`{"account":"some.account@mail.com","listener_running":true,"state_loaded":true,"is_linked":true}`, 0},
		cmdReadyNoAccountBehavior.String():          {`{"listener_running":true,"state_loaded":true,"is_linked":true}`, 0},
		cmdReadyErrorBehavior.String():              {``, 1},
		cmdLoginSuccessBehavior.String():            {`logged in`, 0},
		cmdLoginErrorNoAccountBehavior.String():     {`This email doesn't have a strongDM account.`, 1},
		cmdLoginErrorUnknownBehavior.String():       {`cannot ask for password`, 1},
		cmdLoginInvalidCredentialsBehavior.String(): {`access denied\n`, 1},
		cmdLogoutSuccessBehavior.String():           {`logged out`, 0},
		cmdLogoutNotAuthenticatedBehavior.String():  {`You are not authenticated. Please login again.`, 9},
		cmdLogoutErrorBehavior.String():             {``, 1},
		cmdStatusSuccessBehavior.String():           {`random output`, 0},
		cmdStatusNotAuthenticatedBehavior.String():  {`You are not authenticated. Please login again.`, 9},
		cmdStatusErrorBehavior.String():             {``, 1},
		cmdConnectSuccessBehavior.String():          {`random output`, 0},
		cmdConnectErrorBehavior.String():            {``, 1},
		cmdConnectNotAuthenticatedBehavior.String(): {`You are not authenticated. Please login again.`, 9},
		cmdConnectResourceNotFoundBehavior.String(): {`Cannot find datasource named ''`, 1},
	}

	// Find expected behavior
	if result, ok := outputMap[behavior]; ok {
		fmt.Println(result.output)
		os.Exit(result.exitCode)
	}

	// Unknown behavior
	fmt.Fprintf(os.Stderr, "unknown behavior %q", behavior)
	os.Exit(1)
}

// Helper function to create a test SDMClient
func createTestSDMClient(t *testing.T) *SDMClient {
	testExe, err := os.Executable()
	require.NoError(t, err, "can't determine current executable")

	return NewSDMClient(testExe, WithTimeout(500*time.Millisecond))
}

// Generic test case for all SDM operations
type sdmTestCase struct {
	name            string
	behavior        TestBehavior
	expectedErrMsg  string
	expectedErrCode SDMErrorCode
	shouldError     bool
}

// Helper function to run a test with context
func runWithContext(t *testing.T, tc sdmTestCase, testFn func(context.Context) error) {
	os.Setenv(testSdmBehavior, tc.behavior.String())
	defer os.Unsetenv(testSdmBehavior)

	// Create a context with a reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Run the test function
	err := testFn(ctx)

	if tc.shouldError {
		require.Error(t, err, "Test should return an error")

		if tc.expectedErrMsg != "" {
			assert.Contains(t, err.Error(), tc.expectedErrMsg, "Unexpected error message")
		}

		if tc.expectedErrCode != 0 {
			var sdmErr SDMError
			if errors.As(err, &sdmErr) {
				assert.Equal(t, tc.expectedErrCode, sdmErr.Code, "Unexpected error code")
			} else {
				t.Errorf("Expected SDMError but got different error type: %T", err)
			}
		}
	} else {
		require.NoError(t, err, "Test should not return an error")
	}
}

func TestSDMClient_Ready(t *testing.T) {
	tests := []sdmTestCase{
		{
			name:        "SuccessfulReady",
			behavior:    cmdReadySuccessBehavior,
			shouldError: false,
		},
		{
			name:        "NoAccount",
			behavior:    cmdReadyNoAccountBehavior,
			shouldError: false,
		},
		{
			name:        "Error",
			behavior:    cmdReadyErrorBehavior,
			shouldError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWithContext(t, tc, func(ctx context.Context) error {
				client := createTestSDMClient(t)

				result, err := client.ReadyWithContext(ctx)

				// Additional assertions for Ready-specific results
				if err == nil {
					switch tc.behavior {
					case cmdReadySuccessBehavior:
						assert.NotNil(t, result.Account)
						assert.Equal(t, "some.account@mail.com", *result.Account)
					case cmdReadyNoAccountBehavior:
						assert.Nil(t, result.Account)
					}

					assert.True(t, result.IsLinked)
					assert.True(t, result.StateLoaded)
					assert.True(t, result.ListenerRunning)
				}

				return err
			})
		})
	}
}

func TestSDMClient_Login(t *testing.T) {
	tests := []sdmTestCase{
		{
			name:        "SuccessfulLogin",
			behavior:    cmdLoginSuccessBehavior,
			shouldError: false,
		},
		{
			name:           "ErrorNoAccount",
			behavior:       cmdLoginErrorNoAccountBehavior,
			expectedErrMsg: "This email doesn't have a strongDM account",
			shouldError:    true,
		},
		{
			name:           "ErrorUnknown",
			behavior:       cmdLoginErrorUnknownBehavior,
			expectedErrMsg: "cannot ask for password",
			shouldError:    true,
		},
		{
			name:            "ErrorInvalidCredentials",
			behavior:        cmdLoginInvalidCredentialsBehavior,
			expectedErrMsg:  "access denied",
			expectedErrCode: InvalidCredentials,
			shouldError:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWithContext(t, tc, func(ctx context.Context) error {
				client := createTestSDMClient(t)
				return client.LoginWithContext(ctx, "some.account@mail.com", "password")
			})
		})
	}
}

func TestSDMClient_Logout(t *testing.T) {
	tests := []sdmTestCase{
		{
			name:        "SuccessfulLogout",
			behavior:    cmdLogoutSuccessBehavior,
			shouldError: false,
		},
		{
			name:            "ErrorNotAuthenticated",
			behavior:        cmdLogoutNotAuthenticatedBehavior,
			expectedErrMsg:  "You are not authenticated",
			expectedErrCode: Unauthorized,
			shouldError:     true,
		},
		{
			name:            "ErrorUnknown",
			behavior:        cmdLogoutErrorBehavior,
			expectedErrCode: Unknown,
			shouldError:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWithContext(t, tc, func(ctx context.Context) error {
				client := createTestSDMClient(t)
				return client.LogoutWithContext(ctx)
			})
		})
	}
}

func TestSDMClient_Status(t *testing.T) {
	tests := []sdmTestCase{
		{
			name:        "SuccessfulStatus",
			behavior:    cmdStatusSuccessBehavior,
			shouldError: false,
		},
		{
			name:        "ErrorUnknown",
			behavior:    cmdStatusErrorBehavior,
			shouldError: true,
		},
		{
			name:            "NotAuthenticated",
			behavior:        cmdStatusNotAuthenticatedBehavior,
			expectedErrMsg:  "You are not authenticated",
			expectedErrCode: Unauthorized,
			shouldError:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWithContext(t, tc, func(ctx context.Context) error {
				client := createTestSDMClient(t)
				buf := bytes.NewBuffer(nil)
				err := client.StatusWithContext(ctx, buf)

				// Check output for successful status
				if err == nil && tc.behavior == cmdStatusSuccessBehavior {
					assert.Contains(t, buf.String(), "random output")
				}

				return err
			})
		})
	}
}

func TestSDMClient_Connect(t *testing.T) {
	tests := []sdmTestCase{
		{
			name:        "SuccessfulConnect",
			behavior:    cmdConnectSuccessBehavior,
			shouldError: false,
		},
		{
			name:            "ErrorUnknown",
			behavior:        cmdConnectErrorBehavior,
			expectedErrCode: Unknown,
			shouldError:     true,
		},
		{
			name:            "NotAuthenticated",
			behavior:        cmdConnectNotAuthenticatedBehavior,
			expectedErrMsg:  "You are not authenticated",
			expectedErrCode: Unauthorized,
			shouldError:     true,
		},
		{
			name:            "ResourceNameMissing",
			behavior:        cmdConnectResourceNotFoundBehavior,
			expectedErrMsg:  "Cannot find datasource",
			expectedErrCode: ResourceNotFound,
			shouldError:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWithContext(t, tc, func(ctx context.Context) error {
				client := createTestSDMClient(t)
				return client.ConnectWithContext(ctx, "resource_name")
			})
		})
	}
}

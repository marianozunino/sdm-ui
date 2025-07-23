/*
Copyright © 2025 Mariano Zunino <marianoz@posteo.net>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/marianozunino/sdm-ui/internal/app"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// logOutCmd represents the logout command
var logOutCmd = &cobra.Command{
	Use:     "logout",
	Short:   "Log out of SDM UI",
	Long:    `Logs out of SDM UI by deleting the cache database and forcing a fresh synchronization on next use.`,
	Example: ``,
	Run: func(cmd *cobra.Command, args []string) {
		// Create application instance
		application, err := app.NewApp(
			app.WithAccount(confData.Email),
			app.WithVerbose(confData.Verbose),
			app.WithDbPath(confData.DBPath),
			app.WithCommand(app.DMenuCommandNoop),
			app.WithPasswordCommand(app.PasswordCommandCLI),
			app.WithSSO(confData.UseSSO),
			app.WithTimeout(30*time.Second),
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to initialize application")
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Ensure proper resource cleanup
		defer func() {
			if err := application.Close(); err != nil {
				log.Warn().Err(err).Msg("Error while closing application resources")
			}
		}()

		// Run wipe command with error handling
		if err := application.Logout(); err != nil {
			log.Error().Err(err).Msg("Failed to log out")
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(logOutCmd)

	// Add usage examples to help text
	dmenuCmd.Example = `  # Log out
  sdm-ui logout`
}

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

var (
	useWofi bool
	useRofi bool
)

// dmenuCmd represents the dmenu command
var dmenuCmd = &cobra.Command{
	Use:   "dmenu",
	Short: "Opens dmenu with available data sources",
	Long:  `Displays a menu of available SDM data sources using either rofi or wofi and allows selecting one to connect.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Determine which menu command to use
		var commandOption app.AppOption
		if useWofi {
			commandOption = app.WithCommand(app.DMenuCommandWofi)
			log.Debug().Msg("Using wofi as menu command")
		} else {
			commandOption = app.WithCommand(app.DMenuCommandRofi)
			log.Debug().Msg("Using rofi as menu command")
		}

		// Create application instance
		application, err := app.NewApp(
			app.WithAccount(confData.Email),
			app.WithVerbose(confData.Verbose),
			app.WithDbPath(confData.DBPath),
			app.WithBlacklist(confData.BlacklistPatterns),
			commandOption,
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

		// Run dmenu command with error handling
		if err := application.DMenu(); err != nil {
			log.Error().Err(err).Msg("DMenu operation failed")
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(dmenuCmd)

	// Add menu selection flags
	dmenuCmd.Flags().BoolVarP(&useWofi, "wofi", "w", false, "use wofi as dmenu")
	dmenuCmd.Flags().BoolVarP(&useRofi, "rofi", "r", true, "use rofi as dmenu")

	// Make flags mutually exclusive
	dmenuCmd.MarkFlagsMutuallyExclusive("wofi", "rofi")

	// Add usage examples to help text
	dmenuCmd.Example = `  # Use rofi (default)
  sdm-ui dmenu

  # Use wofi instead
  sdm-ui dmenu --wofi`
}

/*
Copyright © 2024 Mariano Zunino <marianoz@posteo.net>

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
	"github.com/marianozunino/sdm-ui/internal/app"
	"github.com/spf13/cobra"
)

var useWofi bool
var useRofi bool

// dmenuCmd represents the dmenu command
var dmenuCmd = &cobra.Command{
	Use:   "dmenu",
	Short: "Opens dmenu with available data sources",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var commandOption app.AppOption

		if useWofi {
			commandOption = app.WithCommand(app.DMenuCommandWofi)
		} else {
			commandOption = app.WithCommand(app.DMenuCommandRofi)
		}

		app.Newapp(
			app.WithAccount(confData.Email),
			app.WithVerbose(confData.Verbose),
			app.WithDbPath(confData.DBPath),
			app.WithBlacklist(confData.BalcklistPatterns),
			commandOption,
		).DMenu()

	},
}

func init() {
	rootCmd.AddCommand(dmenuCmd)
	dmenuCmd.Flags().BoolVarP(&useWofi, "wofi", "w", false, "use wofi as dmenu")
	dmenuCmd.Flags().BoolVarP(&useRofi, "rofi", "r", true, "use rofi as dmenu")
	// exclusive flags
	dmenuCmd.MarkFlagsMutuallyExclusive("wofi", "rofi")

}

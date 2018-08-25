// Copyright © 2018 Chad Netzer <chad.netzer@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Because the pflags (and flags) boolean options don't toggle the default, but
// instead set it to true when specified, it's best to specify all boolean
// flags with a 'false' default.  So, for options that we want to default to
// true (and thus disable when the option is given), we use a separate flag
// with the opposite default, and toggle it manually after parsing.
type CLIOptions struct {
	StatsOutputDisabled bool
	Options
}

func (c *CLIOptions) NewOptions() Options {
	options := c.Options
	options.StatsOutputEnabled = !c.StatsOutputDisabled
	return options
}

var cfgFile string
var MyCLIOptions CLIOptions

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hardlinkable",
	Short: "A tool to save space by hardlinking identical files",
	Long: `A tool to scan directories and report on the space that could be saved by hard
linking identical files.  It can also perform the linking.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		i, ok := ArgsAreDirs(args)
		if ok {
			Run(args)
		} else {
			fmt.Fprintf(os.Stderr, "'%v' is not a directory.", args[i])
			os.Exit(2)
		}
	},
}

// Return ok if all args are directories, or the index of the first
// non-directory argument
func ArgsAreDirs(args []string) (i int, ok bool) {
	for i, name := range args {
		fi, err := os.Lstat(name)
		if err != nil {
			return i, false
		}
		if !fi.IsDir() {
			return i, false
		}
	}
	return 0, true
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	//rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.hardlinkable.yaml)")

	// Local flags
	flg := rootCmd.Flags()
	var o *CLIOptions = &MyCLIOptions
	flg.CountVarP(&o.Verbosity, "verbose", "v", "Increase verbosity level")
	flg.BoolVar(&o.StatsOutputDisabled, "no-stats", false, "Do not print the final stats")
	flg.BoolVar(&o.ProgressOutputEnabled, "no-progress", false, "Disable progress output while processing")
	flg.BoolVar(&o.JSONOutputEnabled, "json", false, "Output results as JSON")

	flg.BoolVarP(&o.SameName, "same-name", "f", false, "Filenames need to be identical")
	flg.BoolVarP(&o.ContentOnly, "content-only", "c", false, "Only file contents have to match")
	flg.BoolVarP(&o.IgnoreTime, "ignore-time", "t", false, "File modification times need not match")
	flg.BoolVarP(&o.IgnorePerms, "ignore-perms", "p", false, "File permissions need not match")
	flg.BoolVar(&o.IgnoreXattr, "ignore-xattr", false, "Xattrs need not match")

	flg.Uint64VarP(&o.MinFileSize, "min-size", "z", 1, "Minimum file size")
	flg.Uint64VarP(&o.MaxFileSize, "max-size", "Z", 0, "Maximum file size")

	flg.StringP("match", "m", "", "Regular expression used to match files")
	flg.StringP("exclude", "x", "", "Regular expression used to exclude files/dirs")

	// Hidden options
	flg.CountVarP(&o.DebugLevel, "debug", "d", "Increase debugging level")
	flg.MarkHidden("debug")
	flg.IntVarP(&o.LinearSearchThresh, "linear-search-thresh", "", 1, "Length of inode hash lists before switching to digests")
	flg.MarkHidden("linear-search-thresh")
	flg.SortFlags = false
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".hardlinkable" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".hardlinkable")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

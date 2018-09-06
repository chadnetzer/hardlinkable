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
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

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
	StatsOutputDisabled    bool
	ProgressOutputDisabled bool
	CLIMinFileSize         uintN
	CLIMaxFileSize         uintN
	CLIFileIncludes        RegexArray
	CLIFileExcludes        RegexArray
	CLIDirExcludes         RegexArray
	Options
}

func (c *CLIOptions) NewOptions() Options {
	options := c.Options
	options.StatsOutputEnabled = !c.StatsOutputDisabled
	options.ProgressOutputEnabled = !c.ProgressOutputDisabled
	options.MinFileSize = c.CLIMinFileSize.n
	options.MaxFileSize = c.CLIMaxFileSize.n
	options.FileIncludes = c.CLIFileIncludes.vals
	options.FileExcludes = c.CLIFileExcludes.vals
	options.DirExcludes = c.CLIDirExcludes.vals
	return options
}

// Custom pflag Value displays "RE" instead of "stringArray" in usage text
type RegexArray struct {
	flag.Value // "inherit" Value interface
	vals       []string
}

// Return the string "<nil>" to disable default usage text
func (r *RegexArray) String() string {
	return "<nil>"
}

// Implement StringArray Value Set semantics
func (r *RegexArray) Set(val string) error {
	r.vals = append(r.vals, val)
	return nil
}

// Return "RE" instead of "stringArray" for usage text
func (r *RegexArray) Type() string { return "RE" }

// Custom pflag Value displays "N" instead of "uint" in usage text
type uintN struct {
	flag.Value // "inherit" Value interface
	n          uint64
}

// Return the string "0" to disable default usage text
func (u *uintN) String() string {
	return strconv.FormatUint(u.n, 10)
}

// Implement Uint64 humanized Value Set() semantics
func (u *uintN) Set(num string) error {
	var err error
	u.n, err = humanizedUint64(num)
	return err
}

// Return "N" instead of "uint" for usage text
func (u *uintN) Type() string { return "N" }

var cfgFile string
var MyCLIOptions CLIOptions

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "hardlinkable [OPTIONS] dir1 [dir2 ...]",
	Version: "0.9 alpha - 2018-09-05 (Sep 5 2018)",
	Short:   "A tool to save space by hardlinking identical files",
	Long: `A tool to scan directories and report on the space that could be saved by hard
linking identical files.  It can also perform the linking.`,
	Args: cobra.MinimumNArgs(1),
	DisableFlagsInUseLine: true,
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
	flg.CountVarP(&o.Verbosity, "verbose", "v", "``Increase verbosity level (up to 3 times)")
	flg.BoolVar(&o.StatsOutputDisabled, "no-stats", false, "Do not print the final stats")
	flg.BoolVar(&o.ProgressOutputDisabled, "no-progress", false, "Disable progress output while processing")
	flg.BoolVar(&o.JSONOutputEnabled, "json", false, "Output results as JSON")

	flg.BoolVarP(&o.SameName, "same-name", "f", false, "Filenames need to be identical")
	flg.BoolVarP(&o.ContentOnly, "content-only", "c", false, "Only file contents have to match")
	flg.BoolVarP(&o.IgnoreTime, "ignore-time", "t", false, "File modification times need not match")
	flg.BoolVarP(&o.IgnorePerms, "ignore-perms", "p", false, "File permissions need not match")
	flg.BoolVarP(&o.IgnoreOwner, "ignore-owner", "o", false, "File uid/gid need not match")
	flg.BoolVarP(&o.IgnoreXattr, "ignore-xattr", "x", false, "Xattrs need not match")

	o.CLIMinFileSize.n = 1 // default
	flg.VarP(&o.CLIMinFileSize, "min-size", "s", "Minimum file size")
	flg.VarP(&o.CLIMaxFileSize, "max-size", "S", "Maximum file size")

	flg.VarP(&o.CLIFileIncludes, "include", "i", "Regex(es) used to include files (overrides excludes)")
	flg.VarP(&o.CLIFileExcludes, "exclude", "e", "Regex(es) used to exclude files")
	flg.VarP(&o.CLIDirExcludes, "exclude-dir", "E", "Regex(es) used to exclude dirs")

	// Hidden options
	flg.CountVarP(&o.DebugLevel, "debug", "d", "``Increase debugging level")
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

// humanizedUint64 converts humanized size strings like "1k" into an unsigned
// in (ie. 1024)
func humanizedUint64(s string) (uint64, error) {
	s = strings.ToLower(s)
	mult := map[string]uint64{
		"k": 1 << 10, // 1024
		"m": 1 << 20, // 1024**2
		"g": 1 << 30, // 1024**3
		"t": 1 << 40, // 1024**4
		"p": 1 << 50, // 1024**5
	}
	// If the last character is not one of the multiplier letters, try
	// parsing as a normal number string
	c := s[len(s)-1:] // last char
	if _, ok := mult[c]; !ok {
		n, err := strconv.ParseUint(s, 10, 64)
		return n, err
	}
	// Otherwise, parse the prefix digits and apply the multiplier
	n, err := strconv.ParseUint(s[:len(s)-1], 10, 64)
	if err != nil {
		return n, err
	}
	if n > (math.MaxUint64 / mult[c]) {
		return 0, errors.New("Size value is too large for 64 bits")
	}
	return n * mult[c], nil
}

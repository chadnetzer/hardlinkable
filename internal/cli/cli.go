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

package cli

import (
	"errors"
	"flag"
	"fmt"
	"hardlinkable"
	"log"
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
//
// Other cliOptions are converted from one type to another in the Options
// struct
type CLIOptions struct {
	JSONOutputEnabled      bool
	StatsOutputDisabled    bool
	ProgressOutputDisabled bool
	CLIContentOnly         bool
	CLIMinFileSize         uintN
	CLIMaxFileSize         uintN
	CLIFileIncludes        RegexArray
	CLIFileExcludes        RegexArray
	CLIDirExcludes         RegexArray
	CLISearchThresh        intN

	// Verbosity controls the level of output when calling the output
	// options.  Verbosity 0 prints a short summary of results (space
	// saved, etc.). Verbosity 1 outputs additional information on
	// comparison results and other stats.  Verbosity 2 also outputs the
	// linking that would be (or was) performed, and Verbosity 3 prints
	// information on what existing hardlinks were encountered.
	Verbosity int

	hardlinkable.Options
}

func (c CLIOptions) ToOptions() hardlinkable.Options {
	o := c.Options
	o.ShowRunStats = !c.StatsOutputDisabled
	o.MinFileSize = c.CLIMinFileSize.n
	o.MaxFileSize = c.CLIMaxFileSize.n
	o.FileIncludes = c.CLIFileIncludes.vals
	o.FileExcludes = c.CLIFileExcludes.vals
	o.DirExcludes = c.CLIDirExcludes.vals
	o.SearchThresh = c.CLISearchThresh.n
	if c.CLIContentOnly {
		o.IgnoreTime = true
		o.IgnorePerm = true
		o.IgnoreOwner = true
		o.IgnoreXattr = true
	}
	// Verbosity level enables storing new and existing hardlink in
	// Results, as well as the amount of stats output by Results
	if c.Verbosity > 0 {
		o.ShowExtendedRunStats = true
	}
	if c.Verbosity > 1 || c.JSONOutputEnabled {
		o.StoreNewLinkResults = true
	}
	if c.Verbosity > 2 || c.JSONOutputEnabled {
		o.StoreExistingLinkResults = true
	}
	if c.LinkingEnabled {
		c.CheckQuiescence = true
	}
	return o
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

// Custom pflag Value displays "N" instead of "int" in usage text
type intN struct {
	flag.Value // "inherit" Value interface
	n          int
}

// Return the string "0" to disable default usage text
func (i *intN) String() string {
	return strconv.FormatInt(int64(i.n), 10)
}

// Implement Int64 humanized Value Set() semantics
func (i *intN) Set(num string) error {
	N, err := strconv.ParseInt(num, 10, 0)
	if err != nil {
		return err
	}
	i.n = int(N)
	return nil
}

// Return "N" instead of "int" for usage text
func (i *intN) Type() string { return "N" }

type argPaths struct {
	dirs  []string
	files []string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd *cobra.Command
var cfgFile string

// separateArgs will remove duplicate args and separate into dirs and files
func separateArgs(args []string) (argPaths, error) {
	a := argPaths{make([]string, 0), make([]string, 0)}
	seenPaths := make(map[string]struct{}) // key = pathname
	for _, name := range args {
		if _, ok := seenPaths[name]; ok {
			continue
		}
		fi, err := os.Lstat(name)
		if err != nil {
			return a, err
		}
		seenPaths[name] = struct{}{}
		if fi.IsDir() {
			a.dirs = append(a.dirs, name)
			continue
		}
		if fi.Mode().IsRegular() {
			a.files = append(a.files, name)
			continue
		}
		return a, fmt.Errorf("'%v' is neither a directory or a regular file", name)
	}
	return a, nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func CLIRun(dirs []string, files []string, co CLIOptions) {
	var results hardlinkable.Results
	var err error

	opts := co.ToOptions()
	if co.ProgressOutputDisabled {
		results, err = hardlinkable.Run(dirs, files, opts)
	} else {
		results, err = hardlinkable.RunWithProgress(dirs, files, opts)
	}

	if err != nil {
		log.Printf("%v", err)
	}
	if err != nil || !results.RunSuccessful {
		var s string
		switch results.Phase {
		case hardlinkable.StartPhase:
			s = "Stopped before directory walk started.  Results are incomplete..."
		case hardlinkable.WalkPhase:
			s = "Stopped during directory walk.  Results are incomplete..."
		case hardlinkable.LinkPhase:
			if opts.LinkingEnabled {
				s = "Stopped while linking.  Results may be incomplete..."
			} else {
				s = "Stopped while calculating links.  Results may be incomplete..."
			}
		default:
			s = "Stopped early.  Results may be incomplete..."
		}
		fmt.Println(s)
	}

	if co.JSONOutputEnabled {
		results.OutputJSONResults()
	} else {
		results.OutputResults()
	}
}

func init() {
	co := CLIOptions{}

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:     "hardlinkable [OPTIONS] dir1 [dir2...] [files...]",
		Version: "0.9 alpha - 2018-09-05 (Sep 5 2018)",
		Short:   "A tool to save space by hardlinking identical files",
		Long: `A tool to scan directories and report on the space that could be saved
by hard linking identical files.  It can also perform the linking.`,
		Args: cobra.MinimumNArgs(1),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			argP, err := separateArgs(args)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			if co.CLIMaxFileSize.n > 0 && co.CLIMaxFileSize.n < co.CLIMinFileSize.n {
				fmt.Fprintf(os.Stderr,
					"min-size (%v) cannot be larger than max-size (%v)\n",
					co.CLIMinFileSize.n, co.CLIMaxFileSize.n)
				os.Exit(2)
			}
			CLIRun(argP.dirs, argP.files, co)
		},
	}
	cobra.OnInitialize(initConfig)

	//rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.hardlinkable.yaml)")

	// Local flags
	flg := rootCmd.Flags()

	flg.CountVarP(&co.Verbosity, "verbose", "v", "``Increase verbosity level (up to 4 times)")
	flg.BoolVar(&co.StatsOutputDisabled, "no-stats", false, "Do not print the final stats")
	flg.BoolVar(&co.ProgressOutputDisabled, "no-progress", false, "Disable progress output while processing")
	flg.BoolVar(&co.JSONOutputEnabled, "json", false, "Output results as JSON")

	flg.BoolVar(&co.LinkingEnabled, "linking-enabled", false, "Perform the actual linking")

	flg.BoolVarP(&co.SameName, "same-name", "f", false, "Filenames need to be identical")
	flg.BoolVarP(&co.IgnoreTime, "ignore-time", "t", false, "File modification times need not match")
	flg.BoolVarP(&co.IgnorePerm, "ignore-perm", "p", false, "File permission (mode) need not match")
	flg.BoolVarP(&co.IgnoreOwner, "ignore-owner", "o", false, "File uid/gid need not match")
	flg.BoolVarP(&co.IgnoreXattr, "ignore-xattr", "x", false, "Xattrs need not match")
	flg.BoolVarP(&co.CLIContentOnly, "content-only", "c", false, "Only file contents have to match (ie. -potx)")

	co.CLIMinFileSize.n = hardlinkable.DefaultMinFileSize
	flg.VarP(&co.CLIMinFileSize, "min-size", "s", "Minimum file size")
	flg.VarP(&co.CLIMaxFileSize, "max-size", "S", "Maximum file size")

	flg.VarP(&co.CLIFileIncludes, "include", "i", "Regex(es) used to include files (overrides excludes)")
	flg.VarP(&co.CLIFileExcludes, "exclude", "e", "Regex(es) used to exclude files")
	flg.VarP(&co.CLIDirExcludes, "exclude-dir", "E", "Regex(es) used to exclude dirs")
	flg.CountVarP(&co.DebugLevel, "debug", "d", "``Increase debugging level")

	co.CLISearchThresh.n = hardlinkable.DefaultSearchThresh
	flg.VarP(&co.CLISearchThresh, "search-thresh", "", "Ino search length before enabling digests")

	flg.BoolVar(&co.IgnoreLinkingErrors, "ignore-linkerr", false, "Continue even if linking fails")
	flg.BoolVar(&co.CheckQuiescence, "quiescence", false, "Abort if filesystem is being modified")

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

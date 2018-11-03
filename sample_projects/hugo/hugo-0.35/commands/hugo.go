// Copyright 2016 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package commands defines and implements command-line commands and flags
// used by Hugo. Commands and flags are implemented using Cobra.
package commands

import (
	"fmt"
	"io/ioutil"
	"os/signal"
	"sort"
	"sync/atomic"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/gohugoio/hugo/hugofs"

	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	src "github.com/gohugoio/hugo/source"

	"github.com/gohugoio/hugo/config"

	"github.com/gohugoio/hugo/parser"
	flag "github.com/spf13/pflag"

	"regexp"

	"github.com/fsnotify/fsnotify"
	"github.com/gohugoio/hugo/deps"
	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/hugolib"
	"github.com/gohugoio/hugo/livereload"
	"github.com/gohugoio/hugo/utils"
	"github.com/gohugoio/hugo/watcher"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/fsync"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/nitro"
	"github.com/spf13/viper"
)

// Hugo represents the Hugo sites to build. This variable is exported as it
// is used by at least one external library (the Hugo caddy plugin). We should
// provide a cleaner external API, but until then, this is it.
var Hugo *hugolib.HugoSites

const (
	ansiEsc    = "\u001B"
	clearLine  = "\r\033[K"
	hideCursor = ansiEsc + "[?25l"
	showCursor = ansiEsc + "[?25h"
)

// Reset resets Hugo ready for a new full build. This is mainly only useful
// for benchmark testing etc. via the CLI commands.
func Reset() error {
	Hugo = nil
	return nil
}

// commandError is an error used to signal different error situations in command handling.
type commandError struct {
	s         string
	userError bool
}

func (c commandError) Error() string {
	return c.s
}

func (c commandError) isUserError() bool {
	return c.userError
}

func newUserError(a ...interface{}) commandError {
	return commandError{s: fmt.Sprintln(a...), userError: true}
}

func newSystemError(a ...interface{}) commandError {
	return commandError{s: fmt.Sprintln(a...), userError: false}
}

func newSystemErrorF(format string, a ...interface{}) commandError {
	return commandError{s: fmt.Sprintf(format, a...), userError: false}
}

// Catch some of the obvious user errors from Cobra.
// We don't want to show the usage message for every error.
// The below may be to generic. Time will show.
var userErrorRegexp = regexp.MustCompile("argument|flag|shorthand")

func isUserError(err error) bool {
	if cErr, ok := err.(commandError); ok && cErr.isUserError() {
		return true
	}

	return userErrorRegexp.MatchString(err.Error())
}

// HugoCmd is Hugo's root command.
// Every other command attached to HugoCmd is a child command to it.
var HugoCmd = &cobra.Command{
	Use:   "hugo",
	Short: "hugo builds your site",
	Long: `hugo is the main command, used to build your Hugo site.

Hugo is a Fast and Flexible Static Site Generator
built with love by spf13 and friends in Go.

Complete documentation is available at http://gohugo.io/.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		cfgInit := func(c *commandeer) error {
			if buildWatch {
				c.Set("disableLiveReload", true)
			}
			c.Set("renderToMemory", renderToMemory)
			return nil
		}

		c, err := InitializeConfig(buildWatch, cfgInit)
		if err != nil {
			return err
		}

		if buildWatch {
			c.watchConfig()
		}

		return c.build()
	},
}

var hugoCmdV *cobra.Command

// Flags that are to be added to commands.
var (
	buildWatch     bool
	logging        bool
	renderToMemory bool // for benchmark testing
	verbose        bool
	verboseLog     bool
	debug          bool
	quiet          bool
)

var (
	gc              bool
	baseURL         string
	cacheDir        string
	contentDir      string
	layoutDir       string
	cfgFile         string
	destination     string
	logFile         string
	theme           string
	themesDir       string
	source          string
	logI18nWarnings bool
	disableKinds    []string
)

// Execute adds all child commands to the root command HugoCmd and sets flags appropriately.
func Execute() {
	HugoCmd.SetGlobalNormalizationFunc(helpers.NormalizeHugoFlags)

	HugoCmd.SilenceUsage = true

	AddCommands()

	if c, err := HugoCmd.ExecuteC(); err != nil {
		if isUserError(err) {
			c.Println("")
			c.Println(c.UsageString())
		}

		os.Exit(-1)
	}
}

// AddCommands adds child commands to the root command HugoCmd.
func AddCommands() {
	HugoCmd.AddCommand(serverCmd)
	HugoCmd.AddCommand(versionCmd)
	HugoCmd.AddCommand(envCmd)
	HugoCmd.AddCommand(configCmd)
	HugoCmd.AddCommand(checkCmd)
	HugoCmd.AddCommand(benchmarkCmd)
	HugoCmd.AddCommand(convertCmd)
	HugoCmd.AddCommand(newCmd)
	HugoCmd.AddCommand(listCmd)
	HugoCmd.AddCommand(importCmd)

	HugoCmd.AddCommand(genCmd)
	genCmd.AddCommand(genautocompleteCmd)
	genCmd.AddCommand(gendocCmd)
	genCmd.AddCommand(genmanCmd)
	genCmd.AddCommand(createGenDocsHelper().cmd)
	genCmd.AddCommand(createGenChromaStyles().cmd)

}

// initHugoBuilderFlags initializes all common flags, typically used by the
// core build commands, namely hugo itself, server, check and benchmark.
func initHugoBuilderFlags(cmd *cobra.Command) {
	initHugoBuildCommonFlags(cmd)
}

func initRootPersistentFlags() {
	HugoCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is path/config.yaml|json|toml)")
	HugoCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "build in quiet mode")

	// Set bash-completion
	validConfigFilenames := []string{"json", "js", "yaml", "yml", "toml", "tml"}
	_ = HugoCmd.PersistentFlags().SetAnnotation("config", cobra.BashCompFilenameExt, validConfigFilenames)
}

// initHugoBuildCommonFlags initialize common flags related to the Hugo build.
// Called by initHugoBuilderFlags.
func initHugoBuildCommonFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("cleanDestinationDir", false, "remove files from destination not found in static directories")
	cmd.Flags().BoolP("buildDrafts", "D", false, "include content marked as draft")
	cmd.Flags().BoolP("buildFuture", "F", false, "include content with publishdate in the future")
	cmd.Flags().BoolP("buildExpired", "E", false, "include expired content")
	cmd.Flags().StringVarP(&source, "source", "s", "", "filesystem path to read files relative from")
	cmd.Flags().StringVarP(&contentDir, "contentDir", "c", "", "filesystem path to content directory")
	cmd.Flags().StringVarP(&layoutDir, "layoutDir", "l", "", "filesystem path to layout directory")
	cmd.Flags().StringVarP(&cacheDir, "cacheDir", "", "", "filesystem path to cache directory. Defaults: $TMPDIR/hugo_cache/")
	cmd.Flags().BoolP("ignoreCache", "", false, "ignores the cache directory")
	cmd.Flags().StringVarP(&destination, "destination", "d", "", "filesystem path to write files to")
	cmd.Flags().StringVarP(&theme, "theme", "t", "", "theme to use (located in /themes/THEMENAME/)")
	cmd.Flags().StringVarP(&themesDir, "themesDir", "", "", "filesystem path to themes directory")
	cmd.Flags().Bool("uglyURLs", false, "(deprecated) if true, use /filename.html instead of /filename/")
	cmd.Flags().Bool("canonifyURLs", false, "(deprecated) if true, all relative URLs will be canonicalized using baseURL")
	cmd.Flags().StringVarP(&baseURL, "baseURL", "b", "", "hostname (and path) to the root, e.g. http://spf13.com/")
	cmd.Flags().Bool("enableGitInfo", false, "add Git revision, date and author info to the pages")
	cmd.Flags().BoolVar(&gc, "gc", false, "enable to run some cleanup tasks (remove unused cache files) after the build")

	cmd.Flags().BoolVar(&nitro.AnalysisOn, "stepAnalysis", false, "display memory and timing of different steps of the program")
	cmd.Flags().Bool("templateMetrics", false, "display metrics about template executions")
	cmd.Flags().Bool("templateMetricsHints", false, "calculate some improvement hints when combined with --templateMetrics")
	cmd.Flags().Bool("pluralizeListTitles", true, "(deprecated) pluralize titles in lists using inflect")
	cmd.Flags().Bool("preserveTaxonomyNames", false, `(deprecated) preserve taxonomy names as written ("Gérard Depardieu" vs "gerard-depardieu")`)
	cmd.Flags().BoolP("forceSyncStatic", "", false, "copy all files when static is changed.")
	cmd.Flags().BoolP("noTimes", "", false, "don't sync modification time of files")
	cmd.Flags().BoolP("noChmod", "", false, "don't sync permission mode of files")
	cmd.Flags().BoolVarP(&logI18nWarnings, "i18n-warnings", "", false, "print missing translations")

	cmd.Flags().StringSliceVar(&disableKinds, "disableKinds", []string{}, "disable different kind of pages (home, RSS etc.)")

	// Set bash-completion.
	// Each flag must first be defined before using the SetAnnotation() call.
	_ = cmd.Flags().SetAnnotation("source", cobra.BashCompSubdirsInDir, []string{})
	_ = cmd.Flags().SetAnnotation("cacheDir", cobra.BashCompSubdirsInDir, []string{})
	_ = cmd.Flags().SetAnnotation("destination", cobra.BashCompSubdirsInDir, []string{})
	_ = cmd.Flags().SetAnnotation("theme", cobra.BashCompSubdirsInDir, []string{"themes"})
}

func initBenchmarkBuildingFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&renderToMemory, "renderToMemory", false, "render to memory (only useful for benchmark testing)")
}

// init initializes flags.
func init() {
	HugoCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	HugoCmd.PersistentFlags().BoolVarP(&debug, "debug", "", false, "debug output")
	HugoCmd.PersistentFlags().BoolVar(&logging, "log", false, "enable Logging")
	HugoCmd.PersistentFlags().StringVar(&logFile, "logFile", "", "log File path (if set, logging enabled automatically)")
	HugoCmd.PersistentFlags().BoolVar(&verboseLog, "verboseLog", false, "verbose logging")

	initRootPersistentFlags()
	initHugoBuilderFlags(HugoCmd)
	initBenchmarkBuildingFlags(HugoCmd)

	HugoCmd.Flags().BoolVarP(&buildWatch, "watch", "w", false, "watch filesystem for changes and recreate as needed")
	hugoCmdV = HugoCmd

	// Set bash-completion
	_ = HugoCmd.PersistentFlags().SetAnnotation("logFile", cobra.BashCompFilenameExt, []string{})
}

// InitializeConfig initializes a config file with sensible default configuration flags.
func InitializeConfig(running bool, doWithCommandeer func(c *commandeer) error, subCmdVs ...*cobra.Command) (*commandeer, error) {

	var cfg *deps.DepsCfg = &deps.DepsCfg{}

	// Init file systems. This may be changed at a later point.
	osFs := hugofs.Os

	config, err := hugolib.LoadConfig(osFs, source, cfgFile)
	if err != nil {
		return nil, err
	}

	// Init file systems. This may be changed at a later point.
	cfg.Cfg = config

	c, err := newCommandeer(cfg, running)
	if err != nil {
		return nil, err
	}

	for _, cmdV := range append([]*cobra.Command{hugoCmdV}, subCmdVs...) {
		c.initializeFlags(cmdV)
	}

	if baseURL != "" {
		config.Set("baseURL", baseURL)
	}

	if doWithCommandeer != nil {
		if err := doWithCommandeer(c); err != nil {
			return nil, err
		}
	}

	if len(disableKinds) > 0 {
		c.Set("disableKinds", disableKinds)
	}

	logger, err := createLogger(cfg.Cfg)
	if err != nil {
		return nil, err
	}

	cfg.Logger = logger

	config.Set("logI18nWarnings", logI18nWarnings)

	if !config.GetBool("relativeURLs") && config.GetString("baseURL") == "" {
		cfg.Logger.ERROR.Println("No 'baseURL' set in configuration or as a flag. Features like page menus will not work without one.")
	}

	if theme != "" {
		config.Set("theme", theme)
	}

	if themesDir != "" {
		config.Set("themesDir", themesDir)
	}

	if destination != "" {
		config.Set("publishDir", destination)
	}

	var dir string
	if source != "" {
		dir, _ = filepath.Abs(source)
	} else {
		dir, _ = os.Getwd()
	}
	config.Set("workingDir", dir)

	if contentDir != "" {
		config.Set("contentDir", contentDir)
	}

	if layoutDir != "" {
		config.Set("layoutDir", layoutDir)
	}

	if cacheDir != "" {
		config.Set("cacheDir", cacheDir)
	}

	fs := hugofs.NewFrom(osFs, config)

	// Hugo writes the output to memory instead of the disk.
	// This is only used for benchmark testing. Cause the content is only visible
	// in memory.
	if config.GetBool("renderToMemory") {
		fs.Destination = new(afero.MemMapFs)
		// Rendering to memoryFS, publish to Root regardless of publishDir.
		config.Set("publishDir", "/")
	}

	cacheDir = config.GetString("cacheDir")
	if cacheDir != "" {
		if helpers.FilePathSeparator != cacheDir[len(cacheDir)-1:] {
			cacheDir = cacheDir + helpers.FilePathSeparator
		}
		isDir, err := helpers.DirExists(cacheDir, fs.Source)
		utils.CheckErr(cfg.Logger, err)
		if !isDir {
			mkdir(cacheDir)
		}
		config.Set("cacheDir", cacheDir)
	} else {
		config.Set("cacheDir", helpers.GetTempDir("hugo_cache", fs.Source))
	}

	if err := c.initFs(fs); err != nil {
		return nil, err
	}

	cfg.Logger.INFO.Println("Using config file:", config.ConfigFileUsed())

	themeDir := c.PathSpec().GetThemeDir()
	if themeDir != "" {
		if _, err := cfg.Fs.Source.Stat(themeDir); os.IsNotExist(err) {
			return nil, newSystemError("Unable to find theme Directory:", themeDir)
		}
	}

	themeVersionMismatch, minVersion := c.isThemeVsHugoVersionMismatch()

	if themeVersionMismatch {
		cfg.Logger.ERROR.Printf("Current theme does not support Hugo version %s. Minimum version required is %s\n",
			helpers.CurrentHugoVersion.ReleaseVersion(), minVersion)
	}

	return c, nil

}

func createLogger(cfg config.Provider) (*jww.Notepad, error) {
	var (
		logHandle       = ioutil.Discard
		logThreshold    = jww.LevelWarn
		logFile         = cfg.GetString("logFile")
		outHandle       = os.Stdout
		stdoutThreshold = jww.LevelError
	)

	if verboseLog || logging || (logFile != "") {
		var err error
		if logFile != "" {
			logHandle, err = os.OpenFile(logFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
			if err != nil {
				return nil, newSystemError("Failed to open log file:", logFile, err)
			}
		} else {
			logHandle, err = ioutil.TempFile("", "hugo")
			if err != nil {
				return nil, newSystemError(err)
			}
		}
	} else if !quiet && cfg.GetBool("verbose") {
		stdoutThreshold = jww.LevelInfo
	}

	if cfg.GetBool("debug") {
		stdoutThreshold = jww.LevelDebug
	}

	if verboseLog {
		logThreshold = jww.LevelInfo
		if cfg.GetBool("debug") {
			logThreshold = jww.LevelDebug
		}
	}

	// The global logger is used in some few cases.
	jww.SetLogOutput(logHandle)
	jww.SetLogThreshold(logThreshold)
	jww.SetStdoutThreshold(stdoutThreshold)
	helpers.InitLoggers()

	return jww.NewNotepad(stdoutThreshold, logThreshold, outHandle, logHandle, "", log.Ldate|log.Ltime), nil
}

func (c *commandeer) initializeFlags(cmd *cobra.Command) {
	persFlagKeys := []string{"debug", "verbose", "logFile"}
	flagKeys := []string{
		"cleanDestinationDir",
		"buildDrafts",
		"buildFuture",
		"buildExpired",
		"uglyURLs",
		"canonifyURLs",
		"enableRobotsTXT",
		"enableGitInfo",
		"pluralizeListTitles",
		"preserveTaxonomyNames",
		"ignoreCache",
		"forceSyncStatic",
		"noTimes",
		"noChmod",
		"templateMetrics",
		"templateMetricsHints",
	}

	for _, key := range persFlagKeys {
		c.setValueFromFlag(cmd.PersistentFlags(), key)
	}
	for _, key := range flagKeys {
		c.setValueFromFlag(cmd.Flags(), key)
	}

}

var deprecatedFlags = map[string]bool{
	strings.ToLower("uglyURLs"):              true,
	strings.ToLower("pluralizeListTitles"):   true,
	strings.ToLower("preserveTaxonomyNames"): true,
	strings.ToLower("canonifyURLs"):          true,
}

func (c *commandeer) setValueFromFlag(flags *flag.FlagSet, key string) {
	if flags.Changed(key) {
		if _, deprecated := deprecatedFlags[strings.ToLower(key)]; deprecated {
			msg := fmt.Sprintf(`Set "%s = true" in your config.toml.
If you need to set this configuration value from the command line, set it via an OS environment variable: "HUGO_%s=true hugo"`, key, strings.ToUpper(key))
			// Remove in Hugo 0.37
			helpers.Deprecated("hugo", "--"+key+" flag", msg, false)
		}
		f := flags.Lookup(key)
		c.Set(key, f.Value.String())
	}
}

func (c *commandeer) watchConfig() {
	v := c.Cfg.(*viper.Viper)
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		c.Logger.FEEDBACK.Println("Config file changed:", e.Name)
		// Force a full rebuild
		utils.CheckErr(c.Logger, c.recreateAndBuildSites(true))
		if !c.Cfg.GetBool("disableLiveReload") {
			// Will block forever trying to write to a channel that nobody is reading if livereload isn't initialized
			livereload.ForceRefresh()
		}
	})
}

func (c *commandeer) fullBuild() error {
	var (
		g         errgroup.Group
		langCount map[string]uint64
	)

	if !quiet {
		fmt.Print(hideCursor + "Building sites … ")
		defer func() {
			fmt.Print(showCursor + clearLine)
		}()
	}

	copyStaticFunc := func() error {
		cnt, err := c.copyStatic()
		if err != nil {
			return fmt.Errorf("Error copying static files: %s", err)
		}
		langCount = cnt
		return nil
	}
	buildSitesFunc := func() error {
		if err := c.buildSites(); err != nil {
			return fmt.Errorf("Error building site: %s", err)
		}
		return nil
	}
	// Do not copy static files and build sites in parallel if cleanDestinationDir is enabled.
	// This flag deletes all static resources in /public folder that are missing in /static,
	// and it does so at the end of copyStatic() call.
	if c.Cfg.GetBool("cleanDestinationDir") {
		if err := copyStaticFunc(); err != nil {
			return err
		}
		if err := buildSitesFunc(); err != nil {
			return err
		}
	} else {
		g.Go(copyStaticFunc)
		g.Go(buildSitesFunc)
		if err := g.Wait(); err != nil {
			return err
		}
	}

	for _, s := range Hugo.Sites {
		s.ProcessingStats.Static = langCount[s.Language.Lang]
	}

	if gc {
		count, err := Hugo.GC()
		if err != nil {
			return err
		}
		for _, s := range Hugo.Sites {
			// We have no way of knowing what site the garbage belonged to.
			s.ProcessingStats.Cleaned = uint64(count)
		}
	}

	return nil

}

func (c *commandeer) build() error {
	defer c.timeTrack(time.Now(), "Total")

	if err := c.fullBuild(); err != nil {
		return err
	}

	// TODO(bep) Feedback?
	if !quiet {
		fmt.Println()
		Hugo.PrintProcessingStats(os.Stdout)
		fmt.Println()
	}

	if buildWatch {
		watchDirs, err := c.getDirList()
		if err != nil {
			return err
		}
		c.Logger.FEEDBACK.Println("Watching for changes in", c.PathSpec().AbsPathify(c.Cfg.GetString("contentDir")))
		c.Logger.FEEDBACK.Println("Press Ctrl+C to stop")
		watcher, err := c.newWatcher(watchDirs...)
		utils.CheckErr(c.Logger, err)
		defer watcher.Close()

		var sigs = make(chan os.Signal)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		<-sigs
	}

	return nil
}

func (c *commandeer) serverBuild() error {
	defer c.timeTrack(time.Now(), "Total")

	if err := c.fullBuild(); err != nil {
		return err
	}

	// TODO(bep) Feedback?
	if !quiet {
		fmt.Println()
		Hugo.PrintProcessingStats(os.Stdout)
		fmt.Println()
	}

	return nil
}

func (c *commandeer) copyStatic() (map[string]uint64, error) {
	return c.doWithPublishDirs(c.copyStaticTo)
}

func (c *commandeer) createStaticDirsConfig() ([]*src.Dirs, error) {
	var dirsConfig []*src.Dirs

	if !c.languages.IsMultihost() {
		dirs, err := src.NewDirs(c.Fs, c.Cfg, c.DepsCfg.Logger)
		if err != nil {
			return nil, err
		}
		dirsConfig = append(dirsConfig, dirs)
	} else {
		for _, l := range c.languages {
			dirs, err := src.NewDirs(c.Fs, l, c.DepsCfg.Logger)
			if err != nil {
				return nil, err
			}
			dirsConfig = append(dirsConfig, dirs)
		}
	}

	return dirsConfig, nil

}

func (c *commandeer) doWithPublishDirs(f func(dirs *src.Dirs, publishDir string) (uint64, error)) (map[string]uint64, error) {

	langCount := make(map[string]uint64)

	for _, dirs := range c.staticDirsConfig {

		cnt, err := f(dirs, c.pathSpec.PublishDir)
		if err != nil {
			return langCount, err
		}

		if dirs.Language == nil {
			// Not multihost
			for _, l := range c.languages {
				langCount[l.Lang] = cnt
			}
		} else {
			langCount[dirs.Language.Lang] = cnt
		}

	}

	return langCount, nil
}

type countingStatFs struct {
	afero.Fs
	statCounter uint64
}

func (fs *countingStatFs) Stat(name string) (os.FileInfo, error) {
	f, err := fs.Fs.Stat(name)
	if err == nil {
		if !f.IsDir() {
			atomic.AddUint64(&fs.statCounter, 1)
		}
	}
	return f, err
}

func (c *commandeer) copyStaticTo(dirs *src.Dirs, publishDir string) (uint64, error) {

	// If root, remove the second '/'
	if publishDir == "//" {
		publishDir = helpers.FilePathSeparator
	}

	if dirs.Language != nil {
		// Multihost setup.
		publishDir = filepath.Join(publishDir, dirs.Language.Lang)
	}

	staticSourceFs, err := dirs.CreateStaticFs()
	if err != nil {
		return 0, err
	}

	if staticSourceFs == nil {
		c.Logger.WARN.Println("No static directories found to sync")
		return 0, nil
	}

	fs := &countingStatFs{Fs: staticSourceFs}

	syncer := fsync.NewSyncer()
	syncer.NoTimes = c.Cfg.GetBool("noTimes")
	syncer.NoChmod = c.Cfg.GetBool("noChmod")
	syncer.SrcFs = fs
	syncer.DestFs = c.Fs.Destination
	// Now that we are using a unionFs for the static directories
	// We can effectively clean the publishDir on initial sync
	syncer.Delete = c.Cfg.GetBool("cleanDestinationDir")

	if syncer.Delete {
		c.Logger.INFO.Println("removing all files from destination that don't exist in static dirs")

		syncer.DeleteFilter = func(f os.FileInfo) bool {
			return f.IsDir() && strings.HasPrefix(f.Name(), ".")
		}
	}
	c.Logger.INFO.Println("syncing static files to", publishDir)

	// because we are using a baseFs (to get the union right).
	// set sync src to root
	err = syncer.Sync(publishDir, helpers.FilePathSeparator)
	if err != nil {
		return 0, err
	}

	// Sync runs Stat 3 times for every source file (which sounds much)
	numFiles := fs.statCounter / 3

	return numFiles, err
}

func (c *commandeer) timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	c.Logger.FEEDBACK.Printf("%s in %v ms", name, int(1000*elapsed.Seconds()))
}

// getDirList provides NewWatcher() with a list of directories to watch for changes.
func (c *commandeer) getDirList() ([]string, error) {
	var a []string

	// To handle nested symlinked content dirs
	var seen = make(map[string]bool)
	var nested []string

	dataDir := c.PathSpec().AbsPathify(c.Cfg.GetString("dataDir"))
	i18nDir := c.PathSpec().AbsPathify(c.Cfg.GetString("i18nDir"))
	staticSyncer, err := newStaticSyncer(c)
	if err != nil {
		return nil, err
	}

	layoutDir := c.PathSpec().GetLayoutDirPath()
	staticDirs := staticSyncer.d.AbsStaticDirs

	newWalker := func(allowSymbolicDirs bool) func(path string, fi os.FileInfo, err error) error {
		return func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				if path == dataDir && os.IsNotExist(err) {
					c.Logger.WARN.Println("Skip dataDir:", err)
					return nil
				}

				if path == i18nDir && os.IsNotExist(err) {
					c.Logger.WARN.Println("Skip i18nDir:", err)
					return nil
				}

				if path == layoutDir && os.IsNotExist(err) {
					c.Logger.WARN.Println("Skip layoutDir:", err)
					return nil
				}

				if os.IsNotExist(err) {
					for _, staticDir := range staticDirs {
						if path == staticDir && os.IsNotExist(err) {
							c.Logger.WARN.Println("Skip staticDir:", err)
						}
					}
					// Ignore.
					return nil
				}

				c.Logger.ERROR.Println("Walker: ", err)
				return nil
			}

			// Skip .git directories.
			// Related to https://github.com/gohugoio/hugo/issues/3468.
			if fi.Name() == ".git" {
				return nil
			}

			if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
				link, err := filepath.EvalSymlinks(path)
				if err != nil {
					c.Logger.ERROR.Printf("Cannot read symbolic link '%s', error was: %s", path, err)
					return nil
				}
				linkfi, err := helpers.LstatIfOs(c.Fs.Source, link)
				if err != nil {
					c.Logger.ERROR.Printf("Cannot stat %q: %s", link, err)
					return nil
				}
				if !allowSymbolicDirs && !linkfi.Mode().IsRegular() {
					c.Logger.ERROR.Printf("Symbolic links for directories not supported, skipping %q", path)
					return nil
				}

				if allowSymbolicDirs && linkfi.IsDir() {
					// afero.Walk will not walk symbolic links, so wee need to do it.
					if !seen[path] {
						seen[path] = true
						nested = append(nested, path)
					}
					return nil
				}

				fi = linkfi
			}

			if fi.IsDir() {
				if fi.Name() == ".git" ||
					fi.Name() == "node_modules" || fi.Name() == "bower_components" {
					return filepath.SkipDir
				}
				a = append(a, path)
			}
			return nil
		}
	}

	symLinkWalker := newWalker(true)
	regularWalker := newWalker(false)

	// SymbolicWalk will log anny ERRORs
	_ = helpers.SymbolicWalk(c.Fs.Source, dataDir, regularWalker)
	_ = helpers.SymbolicWalk(c.Fs.Source, c.PathSpec().AbsPathify(c.Cfg.GetString("contentDir")), symLinkWalker)
	_ = helpers.SymbolicWalk(c.Fs.Source, i18nDir, regularWalker)
	_ = helpers.SymbolicWalk(c.Fs.Source, layoutDir, regularWalker)
	for _, staticDir := range staticDirs {
		_ = helpers.SymbolicWalk(c.Fs.Source, staticDir, regularWalker)
	}

	if c.PathSpec().ThemeSet() {
		themesDir := c.PathSpec().GetThemeDir()
		_ = helpers.SymbolicWalk(c.Fs.Source, filepath.Join(themesDir, "layouts"), regularWalker)
		_ = helpers.SymbolicWalk(c.Fs.Source, filepath.Join(themesDir, "i18n"), regularWalker)
		_ = helpers.SymbolicWalk(c.Fs.Source, filepath.Join(themesDir, "data"), regularWalker)
	}

	if len(nested) > 0 {
		for {

			toWalk := nested
			nested = nested[:0]

			for _, d := range toWalk {
				_ = helpers.SymbolicWalk(c.Fs.Source, d, symLinkWalker)
			}

			if len(nested) == 0 {
				break
			}
		}
	}

	a = helpers.UniqueStrings(a)
	sort.Strings(a)

	return a, nil
}

func (c *commandeer) recreateAndBuildSites(watching bool) (err error) {
	defer c.timeTrack(time.Now(), "Total")
	if err := c.initSites(); err != nil {
		return err
	}
	if !quiet {
		c.Logger.FEEDBACK.Println("Started building sites ...")
	}
	return Hugo.Build(hugolib.BuildCfg{CreateSitesFromConfig: true})
}

func (c *commandeer) resetAndBuildSites() (err error) {
	if err = c.initSites(); err != nil {
		return
	}
	if !quiet {
		c.Logger.FEEDBACK.Println("Started building sites ...")
	}
	return Hugo.Build(hugolib.BuildCfg{ResetState: true})
}

func (c *commandeer) initSites() error {
	if Hugo != nil {
		return nil
	}
	h, err := hugolib.NewHugoSites(*c.DepsCfg)

	if err != nil {
		return err
	}
	Hugo = h

	return nil
}

func (c *commandeer) buildSites() (err error) {
	if err := c.initSites(); err != nil {
		return err
	}
	return Hugo.Build(hugolib.BuildCfg{})
}

func (c *commandeer) rebuildSites(events []fsnotify.Event) error {
	defer c.timeTrack(time.Now(), "Total")

	if err := c.initSites(); err != nil {
		return err
	}
	visited := c.visitedURLs.PeekAllSet()
	doLiveReload := !buildWatch && !c.Cfg.GetBool("disableLiveReload")
	if doLiveReload && !c.Cfg.GetBool("disableFastRender") {

		// Make sure we always render the home pages
		for _, l := range c.languages {
			langPath := c.PathSpec().GetLangSubDir(l.Lang)
			if langPath != "" {
				langPath = langPath + "/"
			}
			home := c.pathSpec.PrependBasePath("/" + langPath)
			visited[home] = true
		}

	}
	return Hugo.Build(hugolib.BuildCfg{RecentlyVisited: visited}, events...)
}

// newWatcher creates a new watcher to watch filesystem events.
func (c *commandeer) newWatcher(dirList ...string) (*watcher.Batcher, error) {
	if runtime.GOOS == "darwin" {
		tweakLimit()
	}

	staticSyncer, err := newStaticSyncer(c)
	if err != nil {
		return nil, err
	}

	watcher, err := watcher.New(1 * time.Second)

	if err != nil {
		return nil, err
	}

	for _, d := range dirList {
		if d != "" {
			_ = watcher.Add(d)
		}
	}

	go func() {
		for {
			select {
			case evs := <-watcher.Events:
				c.Logger.INFO.Println("Received System Events:", evs)

				staticEvents := []fsnotify.Event{}
				dynamicEvents := []fsnotify.Event{}

				// Special handling for symbolic links inside /content.
				filtered := []fsnotify.Event{}
				for _, ev := range evs {
					// Check the most specific first, i.e. files.
					contentMapped := Hugo.ContentChanges.GetSymbolicLinkMappings(ev.Name)
					if len(contentMapped) > 0 {
						for _, mapped := range contentMapped {
							filtered = append(filtered, fsnotify.Event{Name: mapped, Op: ev.Op})
						}
						continue
					}

					// Check for any symbolic directory mapping.

					dir, name := filepath.Split(ev.Name)

					contentMapped = Hugo.ContentChanges.GetSymbolicLinkMappings(dir)

					if len(contentMapped) == 0 {
						filtered = append(filtered, ev)
						continue
					}

					for _, mapped := range contentMapped {
						mappedFilename := filepath.Join(mapped, name)
						filtered = append(filtered, fsnotify.Event{Name: mappedFilename, Op: ev.Op})
					}
				}

				evs = filtered

				for _, ev := range evs {
					ext := filepath.Ext(ev.Name)
					baseName := filepath.Base(ev.Name)
					istemp := strings.HasSuffix(ext, "~") ||
						(ext == ".swp") || // vim
						(ext == ".swx") || // vim
						(ext == ".tmp") || // generic temp file
						(ext == ".DS_Store") || // OSX Thumbnail
						baseName == "4913" || // vim
						strings.HasPrefix(ext, ".goutputstream") || // gnome
						strings.HasSuffix(ext, "jb_old___") || // intelliJ
						strings.HasSuffix(ext, "jb_tmp___") || // intelliJ
						strings.HasSuffix(ext, "jb_bak___") || // intelliJ
						strings.HasPrefix(ext, ".sb-") || // byword
						strings.HasPrefix(baseName, ".#") || // emacs
						strings.HasPrefix(baseName, "#") // emacs
					if istemp {
						continue
					}
					// Sometimes during rm -rf operations a '"": REMOVE' is triggered. Just ignore these
					if ev.Name == "" {
						continue
					}

					// Write and rename operations are often followed by CHMOD.
					// There may be valid use cases for rebuilding the site on CHMOD,
					// but that will require more complex logic than this simple conditional.
					// On OS X this seems to be related to Spotlight, see:
					// https://github.com/go-fsnotify/fsnotify/issues/15
					// A workaround is to put your site(s) on the Spotlight exception list,
					// but that may be a little mysterious for most end users.
					// So, for now, we skip reload on CHMOD.
					// We do have to check for WRITE though. On slower laptops a Chmod
					// could be aggregated with other important events, and we still want
					// to rebuild on those
					if ev.Op&(fsnotify.Chmod|fsnotify.Write|fsnotify.Create) == fsnotify.Chmod {
						continue
					}

					walkAdder := func(path string, f os.FileInfo, err error) error {
						if f.IsDir() {
							c.Logger.FEEDBACK.Println("adding created directory to watchlist", path)
							if err := watcher.Add(path); err != nil {
								return err
							}
						} else if !staticSyncer.isStatic(path) {
							// Hugo's rebuilding logic is entirely file based. When you drop a new folder into
							// /content on OSX, the above logic will handle future watching of those files,
							// but the initial CREATE is lost.
							dynamicEvents = append(dynamicEvents, fsnotify.Event{Name: path, Op: fsnotify.Create})
						}
						return nil
					}

					// recursively add new directories to watch list
					// When mkdir -p is used, only the top directory triggers an event (at least on OSX)
					if ev.Op&fsnotify.Create == fsnotify.Create {
						if s, err := c.Fs.Source.Stat(ev.Name); err == nil && s.Mode().IsDir() {
							_ = helpers.SymbolicWalk(c.Fs.Source, ev.Name, walkAdder)
						}
					}

					if staticSyncer.isStatic(ev.Name) {
						staticEvents = append(staticEvents, ev)
					} else {
						dynamicEvents = append(dynamicEvents, ev)
					}
				}

				if len(staticEvents) > 0 {
					c.Logger.FEEDBACK.Println("\nStatic file changes detected")
					const layout = "2006-01-02 15:04:05.000 -0700"
					c.Logger.FEEDBACK.Println(time.Now().Format(layout))

					if c.Cfg.GetBool("forceSyncStatic") {
						c.Logger.FEEDBACK.Printf("Syncing all static files\n")
						_, err := c.copyStatic()
						if err != nil {
							utils.StopOnErr(c.Logger, err, "Error copying static files to publish dir")
						}
					} else {
						if err := staticSyncer.syncsStaticEvents(staticEvents); err != nil {
							c.Logger.ERROR.Println(err)
							continue
						}
					}

					if !buildWatch && !c.Cfg.GetBool("disableLiveReload") {
						// Will block forever trying to write to a channel that nobody is reading if livereload isn't initialized

						// force refresh when more than one file
						if len(staticEvents) > 0 {
							for _, ev := range staticEvents {
								path := staticSyncer.d.MakeStaticPathRelative(ev.Name)
								livereload.RefreshPath(path)
							}

						} else {
							livereload.ForceRefresh()
						}
					}
				}

				if len(dynamicEvents) > 0 {
					doLiveReload := !buildWatch && !c.Cfg.GetBool("disableLiveReload")
					onePageName := pickOneWriteOrCreatePath(dynamicEvents)

					c.Logger.FEEDBACK.Println("\nChange detected, rebuilding site")
					const layout = "2006-01-02 15:04:05.000 -0700"
					c.Logger.FEEDBACK.Println(time.Now().Format(layout))

					if err := c.rebuildSites(dynamicEvents); err != nil {
						c.Logger.ERROR.Println("Failed to rebuild site:", err)
					}

					if doLiveReload {
						navigate := c.Cfg.GetBool("navigateToChanged")
						// We have fetched the same page above, but it may have
						// changed.
						var p *hugolib.Page

						if navigate {
							if onePageName != "" {
								p = Hugo.GetContentPage(onePageName)
							}

						}

						if p != nil {
							livereload.NavigateToPathForPort(p.RelPermalink(), p.Site.ServerPort())
						} else {
							livereload.ForceRefresh()
						}
					}
				}
			case err := <-watcher.Errors:
				if err != nil {
					c.Logger.ERROR.Println(err)
				}
			}
		}
	}()

	return watcher, nil
}

func pickOneWriteOrCreatePath(events []fsnotify.Event) string {
	name := ""

	// Some editors (for example notepad.exe on Windows) triggers a change
	// both for directory and file. So we pick the longest path, which should
	// be the file itself.
	for _, ev := range events {
		if (ev.Op&fsnotify.Write == fsnotify.Write || ev.Op&fsnotify.Create == fsnotify.Create) && len(ev.Name) > len(name) {
			name = ev.Name
		}
	}

	return name
}

// isThemeVsHugoVersionMismatch returns whether the current Hugo version is
// less than the theme's min_version.
func (c *commandeer) isThemeVsHugoVersionMismatch() (mismatch bool, requiredMinVersion string) {
	if !c.PathSpec().ThemeSet() {
		return
	}

	themeDir := c.PathSpec().GetThemeDir()

	path := filepath.Join(themeDir, "theme.toml")

	exists, err := helpers.Exists(path, c.Fs.Source)

	if err != nil || !exists {
		return
	}

	b, err := afero.ReadFile(c.Fs.Source, path)

	tomlMeta, err := parser.HandleTOMLMetaData(b)

	if err != nil {
		return
	}

	if minVersion, ok := tomlMeta["min_version"]; ok {
		return helpers.CompareVersion(minVersion) > 0, fmt.Sprint(minVersion)
	}

	return
}

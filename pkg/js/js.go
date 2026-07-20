package js

import (
	_ "embed" // Used to embed helpers.js in the binary.
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/rfc4183"
	"github.com/DNSControl/dnscontrol/v4/pkg/rtypecontrol"
	"github.com/DNSControl/dnscontrol/v4/pkg/transform"
	"github.com/robertkrimen/otto"              // load underscore js into vm by default
	_ "github.com/robertkrimen/otto/underscore" // required by otto
	"github.com/xddxdd/ottoext/fetch"
	"github.com/xddxdd/ottoext/loop"
	"github.com/xddxdd/ottoext/promise"
	"github.com/xddxdd/ottoext/timers"
)

//go:embed helpers.js
var helpersJsStatic string
var helpersJsFileName = "pkg/js/helpers.js"

// currentDirectory is the current directory as used by require().
// This is used to emulate nodejs-style require() directory handling.
// If require("a/b/c.js") is called, any require() statement in c.js
// needs to be accessed relative to "a/b".  Therefore we
// track the currentDirectory (which is the current directory as
// far as require() is concerned, not the actual os.Getwd().
var currentDirectory string

// EnableFetch sets whether to enable fetch() in JS execution environment.
var EnableFetch bool = false

// ExecuteJavaScript accepts a javascript file and runs it, returning the resulting dnsConfig.
func ExecuteJavaScript(file string, devMode bool, variables map[string]string) (*models.DNSConfig, error) {
	script, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	// Record the directory path leading up to this file.
	currentDirectory = filepath.Dir(file)

	return ExecuteJavascriptString(script, devMode, variables)
}

// ExecuteJavascriptString accepts a string containing javascript and runs it, returning the resulting dnsConfig.
func ExecuteJavascriptString(script []byte, devMode bool, variables map[string]string) (*models.DNSConfig, error) {
	vm := otto.New()
	l := loop.New(vm)

	if err := timers.Define(vm, l); err != nil {
		return nil, err
	}
	if err := promise.Define(vm, l); err != nil {
		return nil, err
	}

	// only define fetch() when explicitly enabled
	if EnableFetch {
		if err := fetch.Define(vm, l); err != nil {
			return nil, err
		}
	}

	// add functions to otto
	functions := map[string]any{
		"require":   require,
		"REV":       reverse,
		"REVCOMPAT": reverseCompat,
		"glob":      listFiles, // used for require_glob()
		"PANIC":     jsPanic,
		"HASH":      hashFunc,
	}
	for name, fn := range functions {
		if err := vm.Set(name, fn); err != nil {
			return nil, err
		}
	}

	// add cli variables to otto
	for key, value := range variables {
		if err := vm.Set(key, value); err != nil {
			return nil, err
		}
	}

	helperJs := GetHelpers(devMode)
	// run helper script to prime vm and initialize variables
	if err := l.Eval(helperJs); err != nil {
		return nil, err
	}

	// run user script
	if err := l.Eval(script); err != nil {
		return nil, err
	}

	// wait for event loop to finish
	if err := l.Run(); err != nil {
		return nil, err
	}

	// export conf as string and unmarshal
	value, err := vm.Run(`JSON.stringify(conf)`)
	if err != nil {
		return nil, err
	}
	str, err := value.ToString()
	if err != nil {
		return nil, err
	}
	conf := &models.DNSConfig{}
	if err = json.Unmarshal([]byte(str), conf); err != nil {
		return nil, err
	}

	err = conf.PostProcess()
	if err != nil {
		return nil, err
	}
	// No need to call FixLegacyDC here. These records were created from dnsconfig.js, not from a provider.

	if err := rtypecontrol.ImportRawRecords(conf.Domains); err != nil {
		return nil, err
	}

	return conf, nil
}

// GetHelpers returns the contents of helpers.js, or the embedded version.
func GetHelpers(devMode bool) string {
	if devMode {
		// Load the file:
		b, err := os.ReadFile(helpersJsFileName)
		if err != nil {
			log.Fatal(err)
		}
		return string(b)
	}

	// Return the embedded bytes:
	return helpersJsStatic
}

func require(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 1 {
		throw(call.Otto, "require takes exactly one argument")
	}
	file := call.Argument(0).String() // The filename as given by the user

	// relFile is the file we're actually going to pass to ReadFile().
	// It defaults to the user-provided name unless it is relative.
	relFile := file
	cleanFile := filepath.Clean(filepath.Join(currentDirectory, file))
	if strings.HasPrefix(file, ".") {
		relFile = cleanFile
	}

	// Record the old currentDirectory so that we can return there.
	currentDirectoryOld := currentDirectory
	// Record the directory path leading up to the file we're about to require.
	currentDirectory = filepath.Dir(cleanFile)

	printer.Debugf("requiring: %s (%s)\n", file, relFile)
	// quick fix, by replacing to linux slashes, to make it work with windows paths too.
	data, err := os.ReadFile(filepath.ToSlash(relFile))
	if err != nil {
		throw(call.Otto, err.Error())
	}

	value := otto.TrueValue()

	// If its a json file return the json value, else default to true
	ext := strings.ToLower(filepath.Ext(relFile))
	if strings.HasSuffix(ext, "json") || strings.HasSuffix(ext, "json5") {
		cmd := fmt.Sprintf(`JSON.parse(JSON.stringify(%s))`, string(data))
		value, err = call.Otto.Run(cmd)
	} else {
		_, err = call.Otto.Run(string(data))
	}

	if err != nil {
		throw(call.Otto, fmt.Sprintf("File %s: %s", filepath.Base(relFile), err.Error()))
	}

	// Pop back to the old directory.
	currentDirectory = currentDirectoryOld

	return value
}

func listFiles(call otto.FunctionCall) otto.Value {
	// Check amount of arguments provided
	if len(call.ArgumentList) < 1 || len(call.ArgumentList) > 3 {
		throw(call.Otto, "glob requires at least one argument: folder (string). "+
			"Optional: recursive (bool) [true], fileExtension (string) [.js]")
	}

	// Check if provided parameters are valid
	// First: Let's check dir.
	if !call.Argument(0).IsDefined() || !call.Argument(0).IsString() ||
		len(call.Argument(0).String()) == 0 {
		throw(call.Otto, "glob: first argument needs to be a path, provided as string.")
	}
	dir := call.Argument(0).String() // Path where to start listing
	printer.Debugf("listFiles: cd: %s, user: %s \n", currentDirectory, dir)
	// now we always prepend the current directory we're working in, which is being set within
	// the func ExecuteJavascript() above. So when require("domains/load_all.js") is being used,
	// where glob("customer1/") is being used, we basically search for files in domains/customer1/.
	dir = filepath.ToSlash(filepath.Join(currentDirectory, dir))

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		throw(call.Otto, "glob: provided path does not exist.")
	}

	// Second: Recursive?
	recursive := true
	if call.Argument(1).IsDefined() && !call.Argument(1).IsNull() {
		if call.Argument(1).IsBoolean() {
			recursive, _ = call.Argument(1).ToBoolean() // If it should be recursive
		} else {
			throw(call.Otto, "glob: second argument, if recursive, needs to be bool.")
		}
	}

	// Third: File extension filter.
	fileExtension := ".js"
	if call.Argument(2).IsDefined() && !call.Argument(2).IsNull() {
		if call.Argument(2).IsString() {
			fileExtension = call.Argument(2).String() // Which file extension to filter for.
			if !strings.HasPrefix(fileExtension, ".") {
				// If it doesn't start with a dot, probably user forgot it and we do it instead.
				fileExtension = "." + fileExtension
			}
		} else {
			throw(call.Otto, "glob: third argument, file extension, needs to be a string. * for no filter.")
		}
	}

	// Now we're doing the actual work: Listing files.
	// Folders are ending with a slash. Can be identified later on from the user with JavaScript.
	// Additionally, when more smart logic required, user can use regex in JS.
	files := make([]string, 0)      // init files list
	dirClean := filepath.Clean(dir) // let's clean it here once, instead of over-and-over again within loop
	err := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		// quick fix to get it working on windows, as it returns paths with double-backslash, what usually
		// require() doesn't seem to handle well. For the sake of compatibility (and because slash looks nicer),
		// we simply replace "\\" to "/" using filepath.ToSlash()..
		path = filepath.ToSlash(filepath.Clean(path)) // convert to slashes for directories
		if !recursive && fi.IsDir() {
			// If recursive is disabled, it is a dir what we're processing, and the path is different
			// than specified, we're apparently in a different folder. Therefore: Skip it.
			// So: Why this way? Because Walk() is always recursive and otherwise would require a complete
			// different function to handle this scenario. This way it's easier to maintain.
			if path != dirClean {
				return filepath.SkipDir
			}
		}
		if fileExtension != "*" && fileExtension != filepath.Ext(path) {
			// ONLY skip, when the file extension is NOT matching, or when filter is NOT disabled.
			return nil
		}
		// dirPath := filepath.ToSlash(filepath.Dir(path)) + "/"
		files = append(files, path)
		return err
	})
	if err != nil {
		throw(call.Otto, fmt.Sprintf("dirwalk failed: %v", err.Error()))
	}

	// let's pass the data back to the JS engine.
	value, err := call.Otto.ToValue(files)
	if err != nil {
		throw(call.Otto, fmt.Sprintf("converting value failed: %v", err.Error()))
	}

	return value
}

func jsPanic(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 1 {
		throw(call.Otto, "PANIC takes exactly one argument")
	}

	message := call.Argument(0).String() // The filename as given by the user
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)

	// Won't be actually executed
	v, _ := otto.ToValue(0)
	return v
}

func throw(vm *otto.Otto, str string) {
	panic(vm.MakeCustomError("Error", str))
}

func reverse(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 1 {
		throw(call.Otto, "REV takes exactly one argument")
	}
	dom := call.Argument(0).String()
	rev, err := transform.ReverseDomainName(dom)
	if err != nil {
		throw(call.Otto, err.Error())
	}
	v, _ := otto.ToValue(rev)
	return v
}

func reverseCompat(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 1 {
		throw(call.Otto, "REVCOMPAT takes exactly one argument")
	}
	dom := call.Argument(0).String()
	err := rfc4183.SetCompatibilityMode(dom)
	if err != nil {
		throw(call.Otto, err.Error())
	}
	v, _ := otto.ToValue(nil)
	return v
}

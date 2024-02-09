package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
    "log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	// "github.com/golang/glog"
)

//go:embed version.txt
var versionString string

// ------------------------------------------------------------
// Replaces glog
//
// If the verbosity at the call site is less than or equal to 
// level requested, the log will be enabled.  Higher callsite
// verbosity values are less likely to be output.
//
// if (2 <= verbosity) { log-is-enabled }
//
// Warning: \u261d
// Error: \u274c
// ------------------------------------------------------------
var logger *log.Logger
var verbosity int = 0

func verbose_level(v int) bool {
    return (v <= verbosity)
}

// FindSourceCode scans an input directory recursively for .go files,
// skipping any files or directories specified in exclusions.
func FindSourceCode(inputDirectory string, exclusions map[string]bool) []string {
	paths := []string{}
	// glog.Infof("Scanning %s recursively for .go source", inputDirectory)
	logger.Printf("Scanning %s recursively for .go source", inputDirectory)
	// Files are read in lexical order, i.e. we can later deterministically
	// hash their content: https://pkg.go.dev/path/filepath#WalkDir
	err := filepath.WalkDir(inputDirectory,
		func(path string, info fs.DirEntry, err error) error {
			if err != nil {
				// glog.Errorf("Error %v in directory %s; skipping", err, path)
				logger.Printf("\u274c Error %v in directory %s; skipping", err, path)
				return err
			}

			if b := filepath.Base(path); strings.HasPrefix(b, ".") {
				// glog.V(2).Infof("Skipping %s", path)
                if verbose_level(2) {
                    logger.Printf("Skipping %s", path)
                }
				if info.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			if exclusions[path] {
				if info.IsDir() {
					// glog.Infof("Skipping excluded directory %s and its children", path)
					logger.Printf("Skipping excluded directory %s and its children", path)
					return fs.SkipDir
				}
				// glog.Infof("Skipping excluded file %s", path)
				logger.Printf("Skipping excluded file %s", path)
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			// This is the mandatory format of unit test file names.
			if strings.HasSuffix(path, "_test.go") {
				// glog.V(2).Infof("Skipping test file %s", path)
                if verbose_level(2) {
                    logger.Printf("Skipping test file %s", path)
                }
				return nil
			} else if strings.HasSuffix(path, ".pb.go") {
				// glog.V(1).Infof("Skipping generated file %s", path)
                if verbose_level(1) {
                    logger.Printf("Skipping generated file %s", path)
                }
				return nil
			}

			paths = append(paths, path)

			return nil
		})

	if err != nil {
		// glog.Fatalf("Error walking input directory %s: %v", inputDirectory, err)
		logger.Fatalf("Error walking input directory %s: %v", inputDirectory, err)
	}

	return paths
}

// HashFileContent reads the binary content of
// every file in paths (assumed to be in lexical order)
// and returns the SHA-256 digest.
func HashFileContent(paths []string) string {
	hasher := sha256.New()
	for _, path := range paths {
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			// glog.Fatalf("Error reading file %s: %v", path, err)
			logger.Fatalf("Error reading file %s: %v", path, err)
		}
		hasher.Write(bytes)
	}

	return hex.EncodeToString(hasher.Sum(nil))[0:12]
}

func validateInputAsModule(path string) {
	moduleFile := filepath.Join(path, "go.mod")
	if _, err := os.ReadFile(moduleFile); err != nil {
		// glog.Fatalf("There was no readable go.mod file at %s: %v", path, err)
		logger.Fatalf("There was no readable go.mod file at %s: %v", path, err)
	}
}

func validateAntithesisModule(path string) {
	antithesisModuleFile := filepath.Join(path, "go.mod")
	file, err := os.Open(antithesisModuleFile)
	if err != nil {
		// glog.Fatalf("There was no readable go.mod file at %s: %v", path, err)
		logger.Fatalf("There was no readable go.mod file at %s: %v", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		// glog.Fatalf("There was no readable go.mod file at %s: %v", path, err)
		logger.Fatalf("There was no readable go.mod file at %s: %v", path, err)
	}
	if scanner.Text() != "module antithesis.com/go/instrumentation" {
		// glog.Fatalf("%s does not appear to be the go.mod for the Antithesis wrapper", antithesisModuleFile)
		logger.Fatalf("%s does not appear to be the go.mod for the Antithesis wrapper", antithesisModuleFile)
	}
}

func verifyGoOnPath() {
	// glog.Info("Confirming that go is on $PATH...")
	logger.Printf("Confirming that go is on $PATH...")
	cmd := exec.Command("go", "version")
	_, err := cmd.Output()
	if err != nil {
		// glog.Fatalf("%v", err)
		logger.Fatalf("%v", err)
	}
}

func copyRecursiveNoClobber(from, to string) {
	commandLine := fmt.Sprintf("cp --no-clobber --recursive %s/* %s", from, to)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", commandLine)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// glog.Infof("%s", commandLine)
	logger.Printf("%s", commandLine)
	err := cmd.Run()
	if err != nil {
		// TODO Do something with STDERR?
		// glog.Fatalf("%v", err)
		logger.Fatalf("%v", err)
	}
}

func addDependencies(customerInputDirectory, antithesisOutputDirectory, customerOutputDirectory string) {
	commandLine := fmt.Sprintf("(cd %s; go mod edit -require=antithesis.com/go/instrumentation@v1.0.0 -replace antithesis.com/go/instrumentation=%s -print > %s/go.mod)", customerInputDirectory, antithesisOutputDirectory, customerOutputDirectory)

	cmd := exec.Command("bash", "-c", commandLine)
	// glog.Infof("%s", commandLine)
	logger.Printf("%s", commandLine)
	_, err := cmd.Output()
	if err != nil {
		// Errors here are pretty mysterious.
		// glog.Fatalf("%v", err)
		logger.Fatalf("%v", err)
	}
}

func writeInstrumentedSource(source, path string) error {
	// Any errors here are fatal anyway, so I'm not checking.
	f, e := os.Create(path)
	if e != nil {
		// glog.Warningf("could not create %s", path)
		logger.Printf("\u261d could not create %s", path)
		return e
	}
	defer f.Close()
	_, e = f.WriteString(source)
	if e != nil {
		// glog.Warningf("Could not write instrumented source to %s", path)
		logger.Printf("\u261d Could not write instrumented source to %s", path)
		return e
	}
	return nil
}

func main() {
	versionPtr := flag.Bool("version", false, "the current version of this application")
	exclusionsPtr := flag.String("exclude", "", "the path to a file listing files and directories to exclude from instrumentation (optional)")
	antithesisPtr := flag.String("antithesis", "", "the directory containing the Antithesis instrumentation wrappers (required)")
	prefixPtr := flag.String("prefix", "", "a string to prepend to the symbol table (optional)")
    logfilePtr := flag.String("logfile", "", "file path to log into (default=stderr)")
    verbosePtr := flag.Int("V", 0, "verbosity level (default to 0)")

	flag.Parse()

	if *versionPtr {
		fmt.Println(versionString)
		os.Exit(0)
	}
// Create(name string) (*File, error)
    wrx := os.Stderr
    if logfilePtr != nil {
        if fp, erx := os.Create(*logfilePtr); erx == nil {
            wrx = fp
        }
    }
    logger = log.New(wrx, "", log.LstdFlags | log.Lshortfile)

    if verbosePtr != nil {
        verbosity = *verbosePtr
    }

	if flag.NArg() < 2 {
		flag.Usage()
		fmt.Fprint(os.Stderr, "\nThis program requires 2 positional arguments: an input directory of Golang source to be instrumented, and an output directory for the result.\n")
		os.Exit(1)
	}
	if *antithesisPtr == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *prefixPtr != "" {
		m, _ := regexp.MatchString(`^[a-z]+$`, *prefixPtr)
		if !m {
			fmt.Fprint(os.Stderr, "A prefix must consist of lower-case ASCII letters.")
			os.Exit(1)
		}
	}

	//glog.Info(versionString)
    logger.Println(versionString)

	// defer glog.Flush()

	verifyGoOnPath()

	customerInputDirectory := GetAbsoluteDirectory(flag.Arg(0))
	outputDirectory := GetAbsoluteDirectory(flag.Arg(1))
	ValidateDirectories(customerInputDirectory, outputDirectory)
	validateInputAsModule(customerInputDirectory)
	validateAntithesisModule(*antithesisPtr)

	customerOutputDirectory, antithesisOutputDirectory, symbolsOutputDirectory := createOutputDirectories(outputDirectory)

	exclusions := map[string]bool{}
	if *exclusionsPtr != "" {
		exclusions = ParseExclusionsFile(*exclusionsPtr, customerInputDirectory)
	}

	sourceFiles := FindSourceCode(customerInputDirectory, exclusions)

	hash := HashFileContent(sourceFiles)[0:12]
	// Each module has to have a generated name, and, per Go's rules,
	// be put in a directory with that name.
	shimPkgBase := "instrumented_module_" + hash
	// [PH] shimPkg := InstrumentationModuleName + "/" + shimPkgBase
	shimPkg := InstrumentationModuleName

	shimDirectory := filepath.Join(antithesisOutputDirectory, shimPkgBase)
	shimPath := filepath.Join(shimDirectory, "instrumented_module.go")

	if e := os.MkdirAll(shimDirectory, 0700); e != nil {
		//glog.Fatalf("Could not create subdirectory for Antithesis shim %s: %v", shimPath, e)
		logger.Fatalf("Could not create subdirectory for Antithesis shim %s: %v", shimPath, e)
	}

	// glog.Infof("Instrumenting %s to %s", customerInputDirectory, customerOutputDirectory)
	logger.Printf("Instrumenting %s to %s", customerInputDirectory, customerOutputDirectory)

	symbolTableFileBaseName := "go-" + hash
	if *prefixPtr != "" {
		symbolTableFileBaseName = *prefixPtr + "-" + symbolTableFileBaseName
	}
	symbolTableFileName := symbolTableFileBaseName + ".sym.tsv"
	symbolTableWriter := CreateSymbolTableFile(filepath.Join(symbolsOutputDirectory, symbolTableFileName), symbolTableFileBaseName)

	filesInstrumented := 0

	instrumentor := CreateInstrumentor(customerInputDirectory, shimPkg, symbolTableWriter)

	for _, path := range sourceFiles {
		// glog.Infof("Instrumenting %s", path)
		logger.Printf("Instrumenting %s", path)
		previousEdge := instrumentor.CurrentEdge
		instrumented, e := instrumentor.Instrument(path)

		if e != nil {
			// glog.Errorf("File %s produced error %s; simply copying source", path, e)
			logger.Printf("\u274c File %s produced error %s; simply copying source", path, e)
			continue
		}

		if instrumented == "" {
			// The instrumentor should have reported why it didn't instrument this file.
			continue
		}

		// Strip the prefix from the input file name. We could also use strings.Rel(),
		// but we've got absolute paths, so this will work.
		outputPath := filepath.Join(customerOutputDirectory, path[len(customerInputDirectory):])
		outputSubdirectory := filepath.Dir(outputPath)
		os.MkdirAll(outputSubdirectory, 0755)

		// glog.V(1).Infof("Writing instrumented file %s with edges %d–%d", outputPath, previousEdge, instrumentor.CurrentEdge)
        if verbose_level(1) {
            logger.Printf("Writing instrumented file %s with edges %d–%d", outputPath, previousEdge, instrumentor.CurrentEdge)
        }

		if e = writeInstrumentedSource(instrumented, outputPath); e == nil {
			filesInstrumented++
		}
	}

	if err := symbolTableWriter.Close(); err != nil {
		//glog.Errorf("Could not close symbol table %s: %s", symbolTableWriter.Path, err)
		logger.Printf("\u274c Could not close symbol table %s: %s", symbolTableWriter.Path, err)
	}
	//glog.Infof("Wrote symbol table %s", symbolTableWriter.Path)
	logger.Printf("Wrote symbol table %s", symbolTableWriter.Path)

	writeShimSource(instrumentor.CurrentEdge, shimPkgBase, symbolTableFileName, shimPath)
	// glog.Infof("Antithesis instrumentation shim written to %s", shimPath)
	logger.Printf("Antithesis instrumentation shim written to %s", shimPath)

	copyRecursiveNoClobber(*antithesisPtr, antithesisOutputDirectory)
	// glog.Infof("Antithesis instrumentation module %s copied to %s", *antithesisPtr, antithesisOutputDirectory)
	logger.Printf("Antithesis instrumentation module %s copied to %s", *antithesisPtr, antithesisOutputDirectory)

	addDependencies(customerInputDirectory, antithesisOutputDirectory, customerOutputDirectory)
	// glog.Infof("Antithesis dependencies added to %s/go.mod", customerOutputDirectory)
	logger.Printf("Antithesis dependencies added to %s/go.mod", customerOutputDirectory)

	copyRecursiveNoClobber(customerInputDirectory, customerOutputDirectory)
	// glog.Infof("All other files copied unmodified from %s to %s", customerInputDirectory, customerOutputDirectory)
	logger.Printf("All other files copied unmodified from %s to %s", customerInputDirectory, customerOutputDirectory)

	// glog.Warningf("%d .go files read, %d files skipped, %d edges instrumented", len(sourceFiles), len(sourceFiles)-filesInstrumented, instrumentor.CurrentEdge)
	logger.Printf("\u261d %d .go files read, %d files skipped, %d edges instrumented", len(sourceFiles), len(sourceFiles)-filesInstrumented, instrumentor.CurrentEdge)
}

func writeShimSource(currentEdge int, shimPkg string, symbolTable string, shimPath string) {
	f, err := os.Create(shimPath)
	if err != nil {
		// glog.Fatalf("Could not open wrapper file %s: %v", shimPath, err)
		logger.Fatalf("Could not open wrapper file %s: %v", shimPath, err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	CreateShimSource(shimPkg, symbolTable, currentEdge, w)
	w.Flush()
}

func copyFile(sourcePath string, destinationPath string) {
	inputBytes, e := ioutil.ReadFile(sourcePath)
	e = ioutil.WriteFile(destinationPath, inputBytes, 0644)
	if e != nil {
		// glog.Errorf("Error creating %s: %v", destinationPath, e)
		logger.Printf("\u274c creating %s: %v", destinationPath, e)
	}
}

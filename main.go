package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/term"
)

const (
	minCharactersDefault      = 8
	resultCountDefault        = 10
	exploreHiddenDefault      = false
	extensionsToIgnoreDefault = ".pdf,.png,.jpg,.jpeg,.zip,.mp4,.gif,.ttf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.mp3,.wav,.avi,.mov,.ogg,.wasm,.pyc"
)

// CLI options. Will be initialized by flags
var (
	minCharacters      int      // Minimum number of characters to consider computing entropy
	resultCount        int      // Number of results to display
	exploreHidden      bool     // Ignore hidden files and folders
	extensions         []string // List of file extensions to include. Empty string means all files
	extensionsToIgnore []string // List of file extensions to ignore. Empty string means all files
	discrete           bool     // Discrete mode, don't show the line, only the entropy and file
	includeBinaryFiles bool     // Include binary files in search.
)

type Entropy struct {
	Entropy float64 // Entropy of the line
	File    string  // File where the line is found
	LineNum int     // Line number in the file
	Line    string  // Line with high entropy
}

func NewEntropies(n int) *Entropies {
	return &Entropies{
		Entropies: make([]Entropy, n),
		maxLength: n,
	}
}

// Entropies should be created with NewEntropies(n).
// It should not be written to manually, instead use Entropies.Add
type Entropies struct {
	mu        sync.Mutex
	Entropies []Entropy // Ordered list of entropies with highest entropy first, with length fixed at creation
	maxLength int
}

// Add assumes that es contains an ordered list of entropies of length es.maxLength.
// It preserves ordering, and inserts an additional value e, if it has high enough entropy.
// In that case, the entry with lowest entropy is rejected.
func (es *Entropies) Add(e Entropy) {
	// This condition is to avoid acquiring the lock (slow) if the entropy is not high enough.
	// Not goroutine safe, but another check is made after acquiring the lock.
	if es.Entropies[es.maxLength-1].Entropy >= e.Entropy {
		return
	}

	es.mu.Lock()
	defer es.mu.Unlock()

	if es.Entropies[len(es.Entropies)-1].Entropy >= e.Entropy {
		return
	}

	i, _ := slices.BinarySearchFunc(es.Entropies, e, func(a, b Entropy) int {
		if b.Entropy > a.Entropy {
			return 1
		}
		if a.Entropy > b.Entropy {
			return -1
		}
		return 0
	})

	copy(es.Entropies[i+1:], es.Entropies[i:])
	es.Entropies[i] = e
}

func main() {
	minCharactersFlag := flag.Int("min", minCharactersDefault, "Minimum number of characters in the line to consider computing entropy")
	resultCountFlag := flag.Int("top", resultCountDefault, "Number of results to display")
	exploreHiddenFlag := flag.Bool("include-hidden", exploreHiddenDefault, "Search in hidden files and folders (.git, .env...). Slows down the search.")
	extensionsFlag := flag.String("ext", "", "Search only in files with these extensions. Comma separated list, e.g. -ext go,py,js (default all files)")
	extensionsToIgnoreFlag := flag.String("ignore-ext", "", "Ignore files with these suffixes. Comma separated list, e.g. -ignore-ext min.css,_test.go,pdf,Test.php. Adds ignored extensions to the default ones.")
	noDefaultExtensionsToIgnore := flag.Bool("ignore-ext-no-defaults", false, "Remove the default ignored extensions (default "+extensionsToIgnoreDefault+")")
	discreteFlag := flag.Bool("discrete", false, "Only show the entropy and file, not the line containing the possible secret")
	binaryFilesFlag := flag.Bool("binary", false, "Include binary files in search. Slows down the search and may not be useful. A file is considered binary if the first line is not valid utf8.")

	flag.CommandLine.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s [flags] file1 file2 file3 ...\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Example: %s -top 10 -ext go,py,js .\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Finds the highest entropy strings in files. The higher the entropy, the more random the string is. Useful for finding secrets (and alphabets, it seems).")
		fmt.Fprintln(flag.CommandLine.Output(), "Please support me on GitHub: https://github.com/EwenQuim")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Apply flags
	minCharacters = *minCharactersFlag
	resultCount = *resultCountFlag
	exploreHidden = *exploreHiddenFlag
	discrete = *discreteFlag
	includeBinaryFiles = *binaryFilesFlag
	extensions = strings.Split(*extensionsFlag, ",")
	extensionsToIgnoreString := *extensionsToIgnoreFlag + "," + extensionsToIgnoreDefault
	if *noDefaultExtensionsToIgnore {
		extensionsToIgnoreString = *extensionsToIgnoreFlag
	}
	extensionsToIgnore = strings.Split(extensionsToIgnoreString, ",")
	extensions = removeEmptyStrings(extensions)
	extensionsToIgnore = removeEmptyStrings(extensionsToIgnore)

	// Read file names from cli
	fileNames := flag.Args()
	if len(fileNames) == 0 {
		fmt.Println("No files provided, defaults to current folder.")
		fileNames = []string{"."}
	}
	entropies := NewEntropies(resultCount)
	for _, fileName := range fileNames {
		err := readFile(entropies, fileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", fileName, err)
		}
	}

	redMark := "\033[31m"
	resetMark := "\033[0m"
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		// If not a terminal, remove color
		redMark = ""
		resetMark = ""
	}

	for _, entropy := range entropies.Entropies {
		if entropy == (Entropy{}) {
			return
		}
		if discrete {
			entropy.Line = ""
		}
		fmt.Printf("%.3f: %s%s:%d%s %s\n", entropy.Entropy, redMark, entropy.File, entropy.LineNum, resetMark, entropy.Line)
	}
}

func readFile(entropies *Entropies, fileName string) error {
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		return err
	}

	if isFileHidden(fileInfo.Name()) && !exploreHidden {
		return nil
	}

	if !isFileIncluded(fileInfo.Name()) {
		return nil
	}

	if fileInfo.IsDir() {
		dir, err := os.ReadDir(fileName)
		if err != nil {
			return err
		}

		var wg sync.WaitGroup
		for i, file := range dir {
			wg.Add(1)
			go func(i int, file os.DirEntry) {
				defer wg.Done()
				err := readFile(entropies, fileName+"/"+file.Name())
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file.Name(), err)
				}
			}(i, file)
		}

		wg.Wait()
	}

	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	i := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		i++
		line := strings.TrimSpace(scanner.Text())

		if i == 1 && !includeBinaryFiles && !utf8.ValidString(line) {
			break
		}

		for _, field := range strings.Fields(line) {
			if len(field) < minCharacters {
				continue
			}

			entropies.Add(Entropy{
				Entropy: entropy(field),
				File:    fileName,
				LineNum: i,
				Line:    field,
			})
		}
	}

	return nil
}

func entropy(text string) float64 {
	uniqueCharacters := make(map[rune]int64, len(text))
	for _, r := range text {
		uniqueCharacters[r]++
	}

	entropy := 0.0
	for character := range uniqueCharacters {
		res := float64(uniqueCharacters[character]) / float64(len(text))
		if res == 0 {
			continue
		}

		entropy -= res * math.Log2(res)
	}

	return entropy
}

func isFileHidden(filename string) bool {
	if filename == "." {
		return false
	}
	filename = strings.TrimPrefix(filename, "./")

	return strings.HasPrefix(filename, ".") || filename == "node_modules"
}

// isFileIncluded returns true if the file should be included in the search
func isFileIncluded(filename string) bool {
	for _, ext := range extensionsToIgnore {
		if strings.HasSuffix(filename, ext) {
			return false
		}
	}

	if len(extensions) == 0 {
		return true
	}

	for _, ext := range extensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}

	return false
}

func removeEmptyStrings(slice []string) []string {
	slices.Sort(slice)
	slice = slices.Compact(slice)

	if len(slice) > 0 && slice[0] == "" {
		return slice[1:]
	}

	return slice
}

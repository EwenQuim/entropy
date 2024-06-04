package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

const (
	minCharactersDefault      = 5
	resultCountDefault        = 10
	exploreHiddenDefault      = false
	extensionsToIgnoreDefault = "pdf,png,jpg,jpeg,zip,mp4,gif"
)

var (
	minCharacters      = minCharactersDefault // Minimum number of characters to consider computing entropy
	resultCount        = resultCountDefault   // Number of results to display
	exploreHidden      = exploreHiddenDefault // Ignore hidden files and folders
	extensions         = []string{}           // List of file extensions to include. Empty string means all files
	extensionsToIgnore = []string{}           // List of file extensions to ignore. Empty string means all files
)

type Entropy struct {
	Entropy float64 // Entropy of the line
	File    string  // File where the line is found
	LineNum int     // Line number in the file
	Line    string  // Line with high entropy
}

// Entropies should be created with a size n using make()
// it should not be written to manually, instead use Entropies.Add
type Entropies struct {
	sync.Mutex
	Entropies []Entropy
}

// Add assumes that es contains an ordered set of entropies.
// It preserves ordering, and inserts an additional value e, if it has high enough entropy.
// In that case, the entry with lowest entropy is rejected.
func (es *Entropies) Add(e Entropy) {
	es.Lock()
	defer es.Unlock()

	entropies := es.Entropies
	if entropies[len(entropies)-1].Entropy >= e.Entropy {
		return
	}

	for i := range len(entropies) {
		if entropies[i].Entropy < e.Entropy {
			for j := len(entropies) - 1; j > i; j-- {
				entropies[j] = entropies[j-1]
			}
			entropies[i] = e
			return
		}
	}
}

func main() {
	minCharactersFlag := flag.Int("min", minCharactersDefault, "Minimum number of characters in the line to consider computing entropy")
	resultCountFlag := flag.Int("top", resultCountDefault, "Number of results to display")
	exploreHiddenFlag := flag.Bool("include-hidden", exploreHiddenDefault, "Search in hidden files and folders (.git, .env...). Slows down the search.")
	extensionsFlag := flag.String("ext", "", "Search only in files with these extensions. Comma separated list, e.g. -ext go,py,js (default all files)")
	extensionsToIgnoreFlag := flag.String("ignore-ext", extensionsToIgnoreDefault, "Ignore files with these extensions. Comma separated list, e.g. -ignore-ext pdf,png,jpg")

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
	extensions = strings.Split(*extensionsFlag, ",")
	extensionsToIgnore = strings.Split(*extensionsToIgnoreFlag, ",")

	// Read file names from cli
	fileNames := flag.Args()
	if len(fileNames) == 0 {
		fmt.Println("No files provided, defaults to current folder.")
		fileNames = []string{"."}
	}
	entropies := &Entropies{
		Entropies: make([]Entropy, resultCount),
	}
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
		fmt.Printf("%.2f: %s%s:%d%s %s\n", entropy.Entropy, redMark, entropy.File, entropy.LineNum, resetMark, entropy.Line)
	}
}

func readFile(entropies *Entropies, fileName string) error {
	// If file is a folder, walk inside the folder
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		return err
	}

	if isFileHidden(fileInfo.Name()) && !exploreHidden {
		return nil
	}

	if fileInfo.IsDir() {
		// Walk through the folder and read all files
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

	if !isFileIncluded(fileInfo.Name()) {
		return nil
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

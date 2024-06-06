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
	entropies := make([]Entropy, 0, 10*len(fileNames))
	for _, fileName := range fileNames {
		fileEntropies, err := readFile(fileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", fileName, err)
		}
		entropies = append(entropies, fileEntropies...)
	}

	entropies = sortAndCutTop(entropies)

	redMark := "\033[31m"
	resetMark := "\033[0m"
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		// If not a terminal, remove color
		redMark = ""
		resetMark = ""
	}

	for _, entropy := range entropies {
		fmt.Printf("%.2f: %s%s:%d%s %s\n", entropy.Entropy, redMark, entropy.File, entropy.LineNum, resetMark, entropy.Line)
	}
}

func readFile(fileName string) ([]Entropy, error) {
	// If file is a folder, walk inside the folder
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		return nil, err
	}

	if isFileHidden(fileInfo.Name()) && !exploreHidden {
		return nil, nil
	}

	entropies := make([]Entropy, 0, 10)
	if fileInfo.IsDir() {
		// Walk through the folder and read all files
		dir, err := os.ReadDir(fileName)
		if err != nil {
			return nil, err
		}

		entropiies := make([][]Entropy, len(dir))

		var wg sync.WaitGroup
		for i, file := range dir {
			wg.Add(1)
			go func(i int, file os.DirEntry) {
				defer wg.Done()
				fileEntropies, err := readFile(fileName + "/" + file.Name())
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file.Name(), err)
				}
				entropiies[i] = fileEntropies
			}(i, file)
		}

		wg.Wait()

		for _, fileEntropies := range entropiies {
			entropies = append(entropies, fileEntropies...)
		}
	}

	if !isFileIncluded(fileInfo.Name()) {
		return sortAndCutTop(entropies), nil
	}

	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
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

			entropies = append(entropies, Entropy{
				Entropy: entropy(field),
				File:    fileName,
				LineNum: i,
				Line:    field,
			})
		}
	}

	return sortAndCutTop(entropies), nil
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

func sortAndCutTop(entropies []Entropy) []Entropy {
	slices.SortFunc(entropies, func(a, b Entropy) int {
		return int((b.Entropy - a.Entropy) * 10000)
	})

	if len(entropies) > resultCount {
		return entropies[:resultCount]
	}

	return entropies
}

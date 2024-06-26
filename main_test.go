package main

import (
	"sync"
	"testing"
)

func BenchmarkFile(b *testing.B) {
	entropies := NewEntropies(10)
	for range b.N {
		err := readFile(entropies, "testdata")
		if err != nil {
			b.Errorf("expected nil, got %v", err)
		}
		b.Logf("Entropies: %v", entropies)
	}
}

func TestEntropy(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		Expect(t, entropy(""), 0.0)
	})

	t.Run("single character", func(t *testing.T) {
		Expect(t, entropy("a"), 0.0)
	})

	t.Run("two same characters", func(t *testing.T) {
		Expect(t, entropy("aa"), 0.0)
	})

	t.Run("three different characters", func(t *testing.T) {
		ExpectFloat(t, entropy("abc"), 1.5849625007211563)
	})

	t.Run("three same characters", func(t *testing.T) {
		Expect(t, entropy("aaa"), 0.0)
	})

	t.Run("four different characters", func(t *testing.T) {
		Expect(t, entropy("abcd"), 2.0)
	})

	t.Run("four same characters", func(t *testing.T) {
		Expect(t, entropy("aabb"), 1.0)
	})

	t.Run("12 characters", func(t *testing.T) {
		ExpectFloat(t, entropy("aabbccddeeff"), 2.584962500721156)
	})
}

func TestReadFile(t *testing.T) {
	t.Run("random.js", func(t *testing.T) {
		res := NewEntropies(10)
		err := readFile(res, "testdata/random.js")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}

		ExpectFloat(t, res.Entropies[0].Entropy, 5.53614242151549)
		Expect(t, res.Entropies[0].LineNum, 7) // The token is hidden here
		ExpectFloat(t, res.Entropies[4].Entropy, 3.321928094887362)
	})

	t.Run("testdata/folder", func(t *testing.T) {
		res := NewEntropies(10)
		err := readFile(res, "testdata/folder")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}

		ExpectFloat(t, res.Entropies[0].Entropy, 3.7667029194153567)
		Expect(t, res.Entropies[0].LineNum, 7) // The token is hidden here
		ExpectFloat(t, res.Entropies[6].Entropy, 2.8553885422075336)
	})

	t.Run("dangling symlink in testdata folder", func(t *testing.T) {
		entropies := NewEntropies(10)
		err := readFile(entropies, "testdata")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}

		Expect(t, len(entropies.Entropies), 10)
	})
}

func TestIsFileIncluded(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		extensions = []string{}
		Expect(t, isFileIncluded("main.go"), true)
		Expect(t, isFileIncluded("main.py"), true)
	})

	t.Run("one element included", func(t *testing.T) {
		extensions = []string{"go"}
		Expect(t, isFileIncluded("main.py"), false)
		Expect(t, isFileIncluded("main.go"), true)
	})

	t.Run("one element excluded", func(t *testing.T) {
		extensions = []string{}
		extensionsToIgnore = []string{"go"}
		Expect(t, isFileIncluded("main.go"), false)
		Expect(t, isFileIncluded("main.py"), true)
	})

	t.Run("multiple elements", func(t *testing.T) {
		extensions = []string{"go", "py"}
		extensionsToIgnore = []string{"pdf"}
		Expect(t, isFileIncluded("main.go"), true)
		Expect(t, isFileIncluded("main.py"), true)
		Expect(t, isFileIncluded("main.pdf"), false)
	})
}

func TestIsFileHidden(t *testing.T) {
	Expect(t, isFileHidden("."), false)
	Expect(t, isFileHidden("main.go"), false)
	Expect(t, isFileHidden("main.py"), false)
	Expect(t, isFileHidden("node_modules"), true)
	Expect(t, isFileHidden("./.git"), true)
	Expect(t, isFileHidden("src"), false)
	Expect(t, isFileHidden("./src"), false)
	Expect(t, isFileHidden(".git"), true)
	Expect(t, isFileHidden(".env"), true)
}

func TestEntropies(t *testing.T) {
	t.Run("synchronous", func(t *testing.T) {
		res := NewEntropies(5)
		for _, i := range []float64{1, 3, 5, 7, 2, 4, 6, 8} {
			res.Add(Entropy{Entropy: i})
		}

		Expect(t, res.Entropies[0].Entropy, 8)
		Expect(t, res.Entropies[1].Entropy, 7)
		Expect(t, res.Entropies[2].Entropy, 6)
		Expect(t, res.Entropies[3].Entropy, 5)
		Expect(t, res.Entropies[4].Entropy, 4)
	})

	t.Run("asynchronous (add from multiple parallel goroutines)", func(t *testing.T) {
		res := NewEntropies(5)
		var wg sync.WaitGroup
		for _, i := range []float64{1, 3, 5, 7, 2, 4, 6, 8} {
			wg.Add(1)
			go func(i float64) {
				res.Add(Entropy{Entropy: i})
				wg.Done()
			}(i)
		}
		wg.Wait()
		Expect(t, res.Entropies[0].Entropy, 8)
		Expect(t, res.Entropies[1].Entropy, 7)
		Expect(t, res.Entropies[2].Entropy, 6)
		Expect(t, res.Entropies[3].Entropy, 5)
		Expect(t, res.Entropies[4].Entropy, 4)
	})
}

func Expect[T comparable](t *testing.T, got, expected T) {
	t.Helper()
	if got != expected {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func ExpectFloat(t *testing.T, got, expected float64) {
	t.Helper()

	gotInt := int(got * 10000)
	expectedInt := int(expected * 10000)
	if gotInt != expectedInt {
		t.Errorf("expected %d, got %d", expectedInt, gotInt)
	}
}

func TestRemoveEmptyStrings(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		Expect(t, len(removeEmptyStrings([]string{})), 0)
	})

	t.Run("single empty string", func(t *testing.T) {
		Expect(t, len(removeEmptyStrings([]string{""})), 0)
	})

	t.Run("no empty strings", func(t *testing.T) {
		Expect(t, len(removeEmptyStrings([]string{"a", "b", "c"})), 3)
	})

	t.Run("one empty string", func(t *testing.T) {
		Expect(t, len(removeEmptyStrings([]string{"a", "", "c"})), 2)
	})

	t.Run("multiple consecutive empty strings", func(t *testing.T) {
		Expect(t, len(removeEmptyStrings([]string{"a", "", "", "", "c"})), 2)
	})

	t.Run("multiple non-consecutive empty strings", func(t *testing.T) {
		Expect(t, len(removeEmptyStrings([]string{"", "a", "", "", "", "c", ""})), 2)
	})

	t.Run("all empty strings", func(t *testing.T) {
		Expect(t, len(removeEmptyStrings([]string{"", "", "", ""})), 0)
	})

}

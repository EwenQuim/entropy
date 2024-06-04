package main

import (
	"testing"
)

func BenchmarkFile(b *testing.B) {
	entropies := make(Entropies, 10)
	for range b.N {
		_ = readFile(entropies, "testdata")
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
		res := make(Entropies, 10)
		err := readFile(res, "testdata/random.js")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}

		ExpectFloat(t, res[0].Entropy, 5.53614242151549)
		Expect(t, res[0].LineNum, 7) // The token is hidden here
		ExpectFloat(t, res[4].Entropy, 3.321928094887362)
	})

	t.Run("testdata/folder", func(t *testing.T) {
		res := make(Entropies, 10)
		err := readFile(res, "testdata/folder")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}

		ExpectFloat(t, res[0].Entropy, 3.7667029194153567)
		Expect(t, res[0].LineNum, 7) // The token is hidden here
		ExpectFloat(t, res[6].Entropy, 2.8553885422075336)
	})

	t.Run("dangling symlink in testdata folder", func(t *testing.T) {
		res, err := readFile("testdata")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}

		Expect(t, len(res), 10)
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
	Expect(t, isFileHidden(".git"), true)
	Expect(t, isFileHidden(".env"), true)
}

func TestEntropies(t *testing.T) {
	entropies := make(Entropies, 5)
	for _, i := range []float64{1, 3, 5, 7, 2, 4, 6, 8} {
		entropies.Add(Entropy{Entropy: i})
	}

	Expect(t, entropies[0].Entropy, 8)
	Expect(t, entropies[1].Entropy, 7)
	Expect(t, entropies[2].Entropy, 6)
	Expect(t, entropies[3].Entropy, 5)
	Expect(t, entropies[4].Entropy, 4)
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

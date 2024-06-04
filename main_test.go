package main

import "testing"

func BenchmarkFile(b *testing.B) {
	for range b.N {
		readFile("testdata")
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
		res, err := readFile("testdata/random.js")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}

		Expect(t, len(res), 10)
		ExpectFloat(t, res[0].Entropy, 5.53614242151549)
		Expect(t, res[0].LineNum, 7) // The token is hidden here
		ExpectFloat(t, res[4].Entropy, 3.321928094887362)
	})

	t.Run("testdata/folder", func(t *testing.T) {
		res, err := readFile("testdata/folder")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}

		Expect(t, len(res), 10)
		ExpectFloat(t, res[0].Entropy, 3.7667029194153567)
		Expect(t, res[0].LineNum, 7) // The token is hidden here
		ExpectFloat(t, res[6].Entropy, 2.8553885422075336)
	})
}

func TestSortAndCutTop(t *testing.T) {
	resultCount = 5

	t.Run("nil", func(t *testing.T) {
		res := sortAndCutTop(nil)
		if len(res) != 0 {
			t.Errorf("expected 0, got %d", len(res))
		}
	})

	t.Run("empty", func(t *testing.T) {
		res := sortAndCutTop([]Entropy{})
		if len(res) != 0 {
			t.Errorf("expected 0, got %d", len(res))
		}
	})

	t.Run("less than resultCount", func(t *testing.T) {
		res := sortAndCutTop([]Entropy{
			{Entropy: 0.1},
			{Entropy: 0.6},
			{Entropy: 0.3},
		})

		Expect(t, len(res), 3)
		Expect(t, res[0].Entropy, 0.6)
		Expect(t, res[2].Entropy, 0.1)
	})

	t.Run("more than resultCount", func(t *testing.T) {
		res := sortAndCutTop([]Entropy{
			{Entropy: 0.1},
			{Entropy: 0.6},
			{Entropy: 0.3},
			{Entropy: 0.7},
			{Entropy: 0.4},
			{Entropy: 0.5},
			{Entropy: 0.2},
		})

		Expect(t, len(res), 5)
		Expect(t, res[0].Entropy, 0.7)
		Expect(t, res[4].Entropy, 0.3)
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

![Entropy logo](./entropy.png)

> Paranoïd about having secrets leaked in your huge codebase? Entropy is here to help you find them!

# Entropy

[![Go Reference](https://pkg.go.dev/badge/github.com/EwenQuim/entropy.svg)](https://pkg.go.dev/github.com/EwenQuim/entropy)
[![Go Report Card](https://goreportcard.com/badge/github.com/EwenQuim/entropy)](https://goreportcard.com/report/github.com/EwenQuim/entropy)

Entropy is a CLI tool that will **scan your codebase for high entropy lines**, which are often secrets.

## Installation

### From source with Go (preferred)

```bash
go install github.com/EwenQuim/entropy@latest
entropy

# More options
entropy -h
entropy -top 20 -ext go,py,js
entropy -top 5 -ignore-ext min.js,pdf,png,jpg,jpeg,zip,mp4,gif my-folder my-file1 my-file2
```

or in one line

```bash
go run github.com/EwenQuim/entropy@latest
```

### With brew

```bash
brew install ewenquim/repo/entropy
entropy

# More options
entropy -h
entropy -top 20 -ext go,py,js
entropy -top 5 -ignore-ext min.js,_test.go,pdf,png,jpg my-folder my-file1 my-file2
```

### With docker

```bash
docker run --rm -v $(pwd):/data ewenquim/entropy /data

# More options
docker run --rm -v $(pwd):/data ewenquim/entropy -h
docker run --rm -v $(pwd):/data ewenquim/entropy -top 20 -ext go,py,js /data
docker run --rm -v $(pwd):/data ewenquim/entropy -top 5 /data/my-folder /data/my-file
```

The docker image is available on [Docker Hub](https://hub.docker.com/r/ewenquim/entropy).

The `-v` option is used to mount the current directory into the container. The `/data` directory is the default directory where the tool will look for files. **Don't forget to add /data at the end of the command**, otherwise the tool will search inside the container, not your local filesystem.

## My other projects

- [**Fuego**](https://github.com/go-fuego/fuego): A Go framework that generates OpenAPI documentation from your codebase.
- [**Renpy-Graphviz**](https://github.com/EwenQuim/renpy-graphviz): A tool to generate a graph of the Ren'Py game engine's screens and labels.

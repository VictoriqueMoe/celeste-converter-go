# Celeste Converter (Go)

A Go implementation of a tool for converting between Celeste game assets (DATA format) and PNG images.

## Overview

This tool allows you to convert between Celeste's proprietary DATA format and standard PNG images. It's a Go port of the original [Kotlin-based celeste-converter](https://github.com/borogk/celeste-converter) with parallel processing for improved performance.

## Features

- Convert DATA files to PNG images
- Convert PNG images back to DATA files
- Preserve alpha channel information
- Run-length encoding (RLE) compression support
- Parallel processing for faster batch conversions
- Automatic detection of optimal worker count based on available CPU cores

## Usage

```
celeste-converter [options] [command] <from-directory> <to-directory>
```

Available commands:
- `data2png`: Convert DATA files to PNG images
- `png2data`: Convert PNG images to DATA files

Options:
- `-workers N`: Number of parallel workers (default: number of CPU cores)
- `-verbose`: Enable verbose logging

### Examples

```sh
# Convert all .data files in the "assets" directory to PNG files in the "output" directory
celeste-converter data2png ./assets ./output

# Convert all PNG files back to DATA format with 4 worker threads
celeste-converter -workers 4 png2data ./modified_assets ./output

# Convert with verbose logging
celeste-converter -verbose data2png ./assets ./output
```

## Performance

The parallel processing implementation can significantly speed up conversions when working with large numbers of files. The tool automatically detects the optimal number of worker threads based on your system's CPU cores.

Some performance guidelines:
- For small numbers of files (< 10), parallel processing may not provide significant benefits
- For large batches, the performance scales with the number of CPU cores
- Memory usage increases with the number of workers, so adjust accordingly on memory-constrained systems

## Building from Source

```sh
# Clone the repository
git clone https://github.com/VictoriqueMoe/celeste-converter-go.git
cd celeste-converter-go

# Build the project
go build -o celeste-converter ./cmd/celeste-converter
```
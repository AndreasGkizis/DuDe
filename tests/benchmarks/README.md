# Benchmarks

`BenchmarkHashAndGroupPipeline` runs DuDe's production `CreateHashes` and
`FindDuplicatesInMap` functions over 128 files. Its sub-benchmarks vary the
percentage of files sharing one size from 0% through 100%.

## Metrics

Go runs each benchmark operation repeatedly and reports the average time per
operation. Here, one operation means hashing and grouping the complete 128-file
dataset. Therefore, `2 ms/op` means one full pipeline run averaged two
milliseconds. 

```bash
go test ./tests/benchmarks -run '^$' -bench '^BenchmarkHashAndGroupPipeline$' -benchmem 
```

## Before And After

Capture several samples from the same machine, power profile, Go version, and
worker configuration:

```bash
mkdir -p tests/benchmarks/results

go test ./tests/benchmarks -run '^$' -bench '^BenchmarkHashAndGroupPipeline$' -benchmem -benchtime=1s -count=10 > tests/benchmarks/results/before.txt
```
After implementing an optimization, run the identical command and write to
`after.txt`. Compare both result sets with:

```bash
go install golang.org/x/perf/cmd/benchstat@latest

benchstat tests/benchmarks/results/before.txt tests/benchmarks/results/after.txt
```

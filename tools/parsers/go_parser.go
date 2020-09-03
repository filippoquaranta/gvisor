// Copyright 2020 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package parsers holds parsers to parse Benchmark test ouput.
//
// Parsers parse Benchmark test output and place it in BigQuery
// structs for sending to BigQuery databases.
package parsers

import (
	"fmt"
	"strconv"
	"strings"

	"gvisor.dev/gvisor/test/benchmarks/tools"
	"gvisor.dev/gvisor/tools/bigquery"
)

// parseOutput expects golang benchmark output returns a Benchmark struct formated for BigQuery.
func parseOutput(output string, metadata *bigquery.Metadata, official bool) ([]*bigquery.Benchmark, error) {
	var benchmarks []*bigquery.Benchmark
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if isBenchmark(line) {
			bm, err := parseLine(line, metadata, official)
			if err != nil {
				return nil, fmt.Errorf("failed to parse line '%s': %v", line, err)
			}
			benchmarks = append(benchmarks, bm)
		}
	}
	return benchmarks, nil
}

// isBenchmark checks that a line is a benchmark line with metrics.
func isBenchmark(line string) bool {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return false
	}
	if !strings.HasPrefix(fields[0], "Benchmark") {
		return false
	}
	if _, err := strconv.Atoi(fields[1]); err != nil {
		return false
	}
	return true
}

// parseLine handles parsing a benchmark line into a bigquery.Benchmark.
func parseLine(line string, metadata *bigquery.Metadata, official bool) (*bigquery.Benchmark, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil, fmt.Errorf("two fields required, got: %d", len(fields))
	}

	if !strings.HasPrefix(fields[0], "Benchmark") {
		return nil, fmt.Errorf("invald prefix: %s", fields[0])
	}

	iters, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil, fmt.Errorf("expecting number of runs, got %s: %v", fields[1], err)
	}

	name, params, err := parseNameParams(fields[0])
	if err != nil {
		return nil, fmt.Errorf("parse name/params: %v", err)
	}

	bm := bigquery.NewBenchmark(name, iters, official)
	bm.Metadata = metadata
	for _, p := range params {
		bm.AddCondition(p.Name, p.Value)
	}

	for i := 1; i < len(fields)/2; i++ {
		value := fields[2*i]
		metric := fields[2*i+1]
		if err := makeMetric(bm, value, metric); err != nil {
			return nil, fmt.Errorf("failed on metric %s value: %s:%v", metric, value, err)
		}
	}
	return bm, nil
}

// parseNameParams parses the Name, GOMAXPROCS, and Params from the test.
// field here should be of the format TESTNAME/PARAMS-GOMAXPROCS.
// Parameters will be separated by a "/" with individual params being
// "name.value".
func parseNameParams(field string) (string, []*tools.Parameter, error) {
	var params []*tools.Parameter
	// Remove GOMAXPROCS from end.
	maxIndex := strings.LastIndex(field, "-")
	if maxIndex < 0 {
		return "", nil, fmt.Errorf("GOMAXPROCS not found %s", field)
	}
	maxProcs := field[maxIndex+1:]
	params = append(params, &tools.Parameter{
		Name:  "GOMAXPROCS",
		Value: maxProcs,
	})

	remainder := field[0:maxIndex]
	index := strings.Index(remainder, "/")
	if index == -1 {
		return remainder, params, nil
	}

	name := remainder[0:index]
	p := remainder[index+1:]

	ps, err := tools.NameToParameters(p)
	if err != nil {
		return "", nil, fmt.Errorf("parse params %s: %v", field, err)
	}
	params = append(params, ps...)
	return name, params, nil
}

// makeMetric parses metrics and adds them to the passed Benchmark.
func makeMetric(bm *bigquery.Benchmark, value, metric string) error {
	switch metric {
	// Ignore most output from golang benchmarks.
	case "MB/s":
	case "B/op":
	case "allocs/op":
		return nil
	case "ns/op":
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("ParseFloat %s: %v", value, err)
		}
		bm.AddMetric(metric /*metric name*/, metric /*unit*/, val /*sample*/)
	default:
		m, err := tools.ParseCustomMetric(value, metric)
		if err != nil {
			return fmt.Errorf("failed to parse custom metric %s: %v ", metric, err)
		}
		bm.AddMetric(m.Name, m.Unit, m.Sample)
	}
	return nil
}

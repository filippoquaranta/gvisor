load("//tools:defs.bzl", "go_library")

package(licenses = ["notice"])

go_test(
    name = "parsers_test",
    size = "small",
    srcs = ["go_parser_test.go"],
    library = ":parsers",
    deps = ["//tools/bigquery"],
)

go_library(
    name = "parsers",
    testonly = 1,
    srcs = [
        "go_parser.go",
    ],
    visibility = ["//:sandbox"],
    deps = [
        "//test/benchmarks/tools",
        "//tools/bigquery",
    ],
)

fasttemplate
============

Simple and fast template library for Go.

Fasttemplate peforms only a single task - it substitutes template placeholders
with user-defined values. At high speed :)

Fasttemplate is faster than [text/template](http://golang.org/pkg/text/template/),
[strings.Replace](http://golang.org/pkg/strings/#Replace)
and [strings.Replacer](http://golang.org/pkg/strings/#Replacer) on placeholders'
substitution.

Below are benchmark results comparing fasttemplate performance to text/template,
strings.Replace and strings.Replacer:

```
$ go test -bench=.
PASS
BenchmarkStringsReplace-4            	  500000	      2919 ns/op
BenchmarkStringsReplacer-4           	  500000	      2632 ns/op
BenchmarkTextTemplate-4              	  500000	      3042 ns/op
BenchmarkFastTemplateExecute-4       	 5000000	       328 ns/op
BenchmarkFastTemplateExecuteString-4 	 3000000	       479 ns/op
BenchmarkFastTemplateExecuteTagFunc-4	 2000000	       667 ns/op
```

Docs
====

See http://godoc.org/github.com/valyala/fasttemplate .

Usage
=====

Server:
```go
	template := "http://{{host}}/?q={{query}}&foo={{bar}}{{bar}}"
	t, err := fasttemplate.NewTemplate(template, "{{", "}}")
	if err != nil {
		log.Fatalf("unexpected error when parsing template: %s", err)
	}
	s := t.ExecuteString(map[string]interface{}{
		"host":  "google.com",
		"query": url.QueryEscape("hello=world"),
		"bar":   "foobar",
	})
	fmt.Printf("%s", s)

	// Output:
	// http://google.com/?q=hello%3Dworld&foo=foobarfoobar
```

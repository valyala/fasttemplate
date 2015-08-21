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
$ go test -bench=. -benchmem
PASS
BenchmarkStringsReplace-4               	  500000	      2889 ns/op	    1824 B/op	      14 allocs/op
BenchmarkStringsReplacer-4              	  500000	      2700 ns/op	    2256 B/op	      23 allocs/op
BenchmarkTextTemplate-4                 	  500000	      3089 ns/op	     336 B/op	      19 allocs/op
BenchmarkFastTemplateExecuteFunc-4      	 5000000	       333 ns/op	       0 B/op	       0 allocs/op
BenchmarkFastTemplateExecute-4          	 5000000	       381 ns/op	       0 B/op	       0 allocs/op
BenchmarkFastTemplateExecuteFuncString-4	 3000000	       508 ns/op	     144 B/op	       1 allocs/op
BenchmarkFastTemplateExecuteString-4    	 3000000	       552 ns/op	     144 B/op	       1 allocs/op
BenchmarkFastTemplateExecuteTagFunc-4   	 2000000	       709 ns/op	     144 B/op	       3 allocs/op
```


Docs
====

See http://godoc.org/github.com/valyala/fasttemplate .


Usage
=====

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


Advanced usage
==============

```go
	template := "Hello, [user]! You won [prize]!!! [foobar]"
	t, err := fasttemplate.NewTemplate(template, "[", "]")
	if err != nil {
		log.Fatalf("unexpected error when parsing template: %s", err)
	}
	s := t.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		switch tag {
		case "user":
			return w.Write([]byte("John"))
		case "prize":
			return w.Write([]byte("$100500"))
		default:
			return w.Write([]byte(fmt.Sprintf("[unknown tag %q]", tag)))
		}
	})
	fmt.Printf("%s", s)

	// Output:
	// Hello, John! You won $100500!!! [unknown tag "foobar"]
```

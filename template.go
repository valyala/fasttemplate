// Package fasttemplate implements simple and fast template library.
//
// Fasttemplate is faster than text/template, strings.Replace
// and strings.Replacer.
//
// Fasttemplate ideally fits for fast and simple placeholders' substitutions.
package fasttemplate

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

// Template implements simple template engine, which can be used for fast
// tags' (aka placeholders) substitution.
type Template struct {
	template string
	startTag string
	endTag   string

	texts           [][]byte
	tags            []string
	bytesBufferPool sync.Pool
}

// New parses the given template using the given startTag and endTag
// as tag start and tag end.
//
// The returned template can be executed by concurrently running goroutines
// using Execute* methods.
//
// New panics if the given template cannot be parsed. Use NewTemplate instead
// if template may contain errors.
func New(template, startTag, endTag string) *Template {
	t, err := NewTemplate(template, startTag, endTag)
	if err != nil {
		panic(err)
	}
	return t
}

// NewTemplate parses the given template using the given startTag and endTag
// as tag start and tag end.
//
// The returned template can be executed by concurrently running goroutines
// using Execute* methods.
func NewTemplate(template, startTag, endTag string) (*Template, error) {
	var t Template
	err := t.Reset(template, startTag, endTag)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func newBytesBuffer() interface{} {
	return &bytes.Buffer{}
}

// TagFunc can be used as a substitution value in the map passed to Execute*.
// Execute* functions pass tag (placeholder) name in 'tag' argument.
//
// TagFunc must be safe to call from concurrently running goroutines.
//
// TagFunc must write contents to w and return the number of bytes written.
type TagFunc func(w io.Writer, tag string) (int, error)

// Reset resets the template t to new one defined by
// template, startTag and endTag.
//
// Reset allows Template object re-use.
//
// Reset may be called only if no other goroutines call t methods at the moment.
func (t *Template) Reset(template, startTag, endTag string) error {
	// Keep these vars in t, so GC won't collect them and won't break
	// vars derived via unsafe*
	t.bytesBufferPool.New = newBytesBuffer
	t.template = template
	t.startTag = startTag
	t.endTag = endTag
	t.texts = t.texts[:0]
	t.tags = t.tags[:0]

	if len(startTag) == 0 {
		panic("startTag cannot be empty")
	}
	if len(endTag) == 0 {
		panic("endTag cannot be empty")
	}

	s := unsafeString2Bytes(template)
	a := unsafeString2Bytes(startTag)
	b := unsafeString2Bytes(endTag)

	tagsCount := bytes.Count(s, a)
	if tagsCount == 0 {
		return nil
	}

	if tagsCount+1 > cap(t.texts) {
		t.texts = make([][]byte, 0, tagsCount+1)
	}
	if tagsCount > cap(t.tags) {
		t.tags = make([]string, 0, tagsCount)
	}

	for {
		n := bytes.Index(s, a)
		if n < 0 {
			t.texts = append(t.texts, s)
			break
		}
		t.texts = append(t.texts, s[:n])

		s = s[n+len(a):]
		n = bytes.Index(s, b)
		if n < 0 {
			return fmt.Errorf("Cannot find end tag=%q in the template=%q starting from %q", endTag, template, s)
		}

		t.tags = append(t.tags, unsafeBytes2String(s[:n]))
		s = s[n+len(b):]
	}

	return nil
}

// ExecuteFunc calls f on each template tag (placeholder) occurrence.
//
// Returns the number of bytes written to w.
func (t *Template) ExecuteFunc(w io.Writer, f TagFunc) (int64, error) {
	var nn int64

	n := len(t.texts) - 1
	if n == -1 {
		ni, err := w.Write(unsafeString2Bytes(t.template))
		return int64(ni), err
	}

	for i := 0; i < n; i++ {
		ni, err := w.Write(t.texts[i])
		if err != nil {
			return nn, err
		}
		nn += int64(ni)

		if ni, err = f(w, t.tags[i]); err != nil {
			return nn, err
		}
		nn += int64(ni)
	}
	ni, err := w.Write(t.texts[n])
	if err != nil {
		return nn, err
	}
	nn += int64(ni)
	return nn, nil
}

// Execute substitutes template tags (placeholders) with the corresponding
// values from the map m and writes the result to the given writer w.
//
// Substitution map m may contain values with the following types:
//   * []byte - the fastest value type
//   * string - convenient value type
//   * TagFunc - flexible value type
//
// Returns the number of bytes written to w.
func (t *Template) Execute(w io.Writer, m map[string]interface{}) (int64, error) {
	return t.ExecuteFunc(w, func(w io.Writer, tag string) (int, error) { return stdTagFunc(w, tag, m) })
}

// ExecuteFuncString call f on each template tag (placeholder) occurrence
// and substitutes it with the data written to TagFunc's w.
//
// Returns the resulting string.
func (t *Template) ExecuteFuncString(f TagFunc) string {
	w := t.bytesBufferPool.Get().(*bytes.Buffer)
	if _, err := t.ExecuteFunc(w, f); err != nil {
		panic(fmt.Sprintf("unexpected error: %s", err))
	}
	s := string(w.Bytes())
	w.Reset()
	t.bytesBufferPool.Put(w)
	return s
}

// ExecuteString substitutes template tags (placeholders) with the corresponding
// values from the map m and returns the result.
//
// Substitution map m may contain values with the following types:
//   * []byte - the fastest value type
//   * string - convenient value type
//   * TagFunc - flexible value type
//
func (t *Template) ExecuteString(m map[string]interface{}) string {
	return t.ExecuteFuncString(func(w io.Writer, tag string) (int, error) { return stdTagFunc(w, tag, m) })
}

func stdTagFunc(w io.Writer, tag string, m map[string]interface{}) (int, error) {
	v := m[tag]
	if v == nil {
		return 0, nil
	}
	switch value := v.(type) {
	case []byte:
		return w.Write(value)
	case string:
		return w.Write([]byte(value))
	case TagFunc:
		return value(w, tag)
	default:
		panic(fmt.Sprintf("tag=%q contains unexpected value type=%#v. Expected []byte, string or TagFunc", tag, v))
	}
}

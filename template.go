package fasttemplate

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

// Template implements simple template engine, which can be used for fast
// tags (aka placeholders) substitution.
type Template struct {
	texts           [][]byte
	tags            []string
	bytesBufferPool sync.Pool
}

// NewTemplate parses the given template using the given startTag and endTag
// as tag start and tag end.
//
// The returned template can be executed by concurrently running goroutines
// using Execute* methods.
func NewTemplate(template, startTag, endTag string) (*Template, error) {
	var t Template

	if len(startTag) == 0 {
		panic("startTag cannot be empty")
	}
	if len(endTag) == 0 {
		panic("endTag cannot be empty")
	}

	s := []byte(template)
	a := []byte(startTag)
	b := []byte(endTag)

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
			return nil, fmt.Errorf("Cannot find end tag=%q in the template=%q starting from %q", endTag, template, s)
		}

		t.tags = append(t.tags, string(s[:n]))
		s = s[n+len(b):]
	}

	return &t, nil
}

// TagFunc can be used as a substitution value in the map passed to Execute*.
//
// It must write contents to w and return the number of bytes written.
type TagFunc func(w io.Writer) (int, error)

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
	var nn int64

	n := len(t.texts) - 1
	for i := 0; i < n; i++ {
		ni, err := w.Write(t.texts[i])
		if err != nil {
			return nn, err
		}
		nn += int64(ni)

		k := t.tags[i]
		v := m[k]
		if v == nil {
			continue
		}
		switch value := v.(type) {
		case []byte:
			ni, err = w.Write(value)
		case string:
			ni, err = w.Write([]byte(value))
		case TagFunc:
			ni, err = value(w)
		default:
			panic(fmt.Sprintf("key=%q contains unexpected value type=%#v. Expected []byte, string or TagFunc", k, v))
		}
		if err != nil {
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

// ExecuteString substitutes template tags (placeholders) with the corresponding
// values from the map m and returns the result.
//
// Substitution map m may contain values with the following types:
//   * []byte - the fastest value type
//   * string - convenient value type
//   * TagFunc - flexible value type
//
func (t *Template) ExecuteString(m map[string]interface{}) string {
	var w *bytes.Buffer
	wv := t.bytesBufferPool.Get()
	if wv == nil {
		w = &bytes.Buffer{}
	} else {
		w = wv.(*bytes.Buffer)
	}
	_, err := t.Execute(w, m)
	if err != nil {
		panic(fmt.Sprintf("unexpected error: %s", err))
	}
	s := string(w.Bytes())
	w.Reset()
	t.bytesBufferPool.Put(w)
	return s
}

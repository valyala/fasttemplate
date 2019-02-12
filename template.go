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
	"reflect"

	"github.com/valyala/bytebufferpool"
)

// ExecuteFunc calls f on each template tag (placeholder) occurrence.
//
// Returns the number of bytes written to w.
//
// This function is optimized for constantly changing templates.
// Use Template.ExecuteFunc for frozen templates.
func ExecuteFunc(template, startTag, endTag string, w io.Writer, f TagFunc) (int64, error) {
	s := unsafeString2Bytes(template)
	a := unsafeString2Bytes(startTag)
	b := unsafeString2Bytes(endTag)

	var nn int64
	var ni int
	var err error
	for {
		n := bytes.Index(s, a)
		if n < 0 {
			break
		}
		ni, err = w.Write(s[:n])
		nn += int64(ni)
		if err != nil {
			return nn, err
		}

		s = s[n+len(a):]
		n = bytes.Index(s, b)
		if n < 0 {
			// cannot find end tag - just write it to the output.
			ni, _ = w.Write(a)
			nn += int64(ni)
			break
		}

		ni, err = f(w, unsafeBytes2String(s[:n]))
		nn += int64(ni)
		s = s[n+len(b):]
	}
	ni, err = w.Write(s)
	nn += int64(ni)

	return nn, err
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
//
// This function is optimized for constantly changing templates.
// Use Template.Execute for frozen templates.
func Execute(template, startTag, endTag string, w io.Writer, m map[string]interface{}) (int64, error) {
	return ExecuteFunc(template, startTag, endTag, w, func(w io.Writer, tag string) (int, error) { return stdTagFunc(w, tag, m) })
}

// ExecuteFuncString calls f on each template tag (placeholder) occurrence
// and substitutes it with the data written to TagFunc's w.
//
// Returns the resulting string.
//
// This function is optimized for constantly changing templates.
// Use Template.ExecuteFuncString for frozen templates.
func ExecuteFuncString(template, startTag, endTag string, f TagFunc) (string, error) {
	var s string

	tagsCount := bytes.Count(unsafeString2Bytes(template), unsafeString2Bytes(startTag))
	if tagsCount == 0 {
		return template, nil
	}

	bb := byteBufferPool.Get()
	if _, err := ExecuteFunc(template, startTag, endTag, bb, f); err != nil {
		return s, err
	}
	s = string(bb.B)
	bb.Reset()
	byteBufferPool.Put(bb)
	return s, nil
}

var byteBufferPool bytebufferpool.Pool

// ExecuteString substitutes template tags (placeholders) with the corresponding
// values from the map m and returns the result.
//
// Substitution map m may contain values with the following types:
//   * []byte - the fastest value type
//   * string - convenient value type
//   * TagFunc - flexible value type
//
// This function is optimized for constantly changing templates.
// Use Template.ExecuteString for frozen templates.
func ExecuteString(template, startTag, endTag string, m map[string]interface{}) (string, error) {
	return ExecuteFuncString(template, startTag, endTag, func(w io.Writer, tag string) (int, error) { return stdTagFunc(w, tag, m) })
}

// Template implements simple template engine, which can be used for fast
// tags' (aka placeholders) substitution.
type Template struct {
	template string
	startTag string
	endTag   string

	texts          [][]byte
	tags           []string
	byteBufferPool bytebufferpool.Pool
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
	t.template = template
	t.startTag = startTag
	t.endTag = endTag
	t.texts = t.texts[:0]
	t.tags = t.tags[:0]

	if len(startTag) == 0 {
		return ErrEmptyStartTag
	}
	if len(endTag) == 0 {
		return ErrEmptyEndTag
	}

	templateBytes := unsafeString2Bytes(template)
	startTagBytes := unsafeString2Bytes(startTag)
	endTagBytes := unsafeString2Bytes(endTag)

	tagsCount := bytes.Count(templateBytes, startTagBytes)
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
		n := bytes.Index(templateBytes, startTagBytes)
		if n < 0 {
			t.texts = append(t.texts, templateBytes)
			break
		}
		t.texts = append(t.texts, templateBytes[:n])

		templateBytes = templateBytes[n+len(startTagBytes):]

		startTagIdx := bytes.Index(templateBytes, startTagBytes)
		endTagIdx := bytes.Index(templateBytes, endTagBytes)
		var missingTag []byte
		for (startTagIdx < endTagIdx) && (startTagIdx > -1) {
			missingTag = append(missingTag, templateBytes[:startTagIdx+len(startTagBytes)]...)
			templateBytes = templateBytes[startTagIdx+len(startTagBytes):]
			startTagIdx = bytes.Index(templateBytes, startTagBytes)
			endTagIdx = bytes.Index(templateBytes, endTagBytes)
		}

		nNext := bytes.Index(templateBytes, startTagBytes)
		if nNext < 0 {
			nNext = len(templateBytes)
		}

		if reflect.DeepEqual(startTagBytes, endTagBytes) {
			sRemaining := templateBytes[nNext+len(startTagBytes):]

			nNextNext := secondIndex(sRemaining, startTagBytes)
			if nNextNext < 0 {
				nNext = len(templateBytes)
			} else {
				nNext = nNextNext
			}
		}

		n = bytes.LastIndex(templateBytes[:nNext], endTagBytes)
		if n < 0 {
			return fmt.Errorf("cannot find end tag=%q in the template=%q starting from %q", endTag, template, templateBytes)
		}

		tag := append(missingTag, templateBytes[:n]...)
		t.tags = append(t.tags, unsafeBytes2String(bytes.TrimSpace(tag)))
		templateBytes = templateBytes[n+len(endTagBytes):]
	}
	return nil
}

// ExecuteFunc calls f on each template tag (placeholder) occurrence.
//
// Returns the number of bytes written to w.
//
// This function is optimized for frozen templates.
// Use ExecuteFunc for constantly changing templates.
func (t *Template) ExecuteFunc(w io.Writer, f TagFunc) (int64, error) {
	var nn int64

	n := len(t.texts) - 1
	if n == -1 {
		ni, err := w.Write(unsafeString2Bytes(t.template))
		return int64(ni), err
	}

	for i := 0; i < n; i++ {
		ni, err := w.Write(t.texts[i])
		nn += int64(ni)
		if err != nil {
			return nn, err
		}

		ni, err = f(w, t.tags[i])
		nn += int64(ni)
		if err != nil {
			return nn, err
		}
	}
	ni, err := w.Write(t.texts[n])
	nn += int64(ni)
	return nn, err
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

// ExecuteFuncString calls f on each template tag (placeholder) occurrence
// and substitutes it with the data written to TagFunc's w.
//
// Returns the resulting string.
//
// This function is optimized for frozen templates.
// Use ExecuteFuncString for constantly changing templates.
func (t *Template) ExecuteFuncString(f TagFunc) (string, error) {
	var s string
	bb := t.byteBufferPool.Get()
	if _, err := t.ExecuteFunc(bb, f); err != nil {
		return s, err
	}
	s = string(bb.Bytes())
	bb.Reset()
	t.byteBufferPool.Put(bb)
	return s, nil
}

// ExecuteString substitutes template tags (placeholders) with the corresponding
// values from the map m and returns the result.
//
// Substitution map m may contain values with the following types:
//   * []byte - the fastest value type
//   * string - convenient value type
//   * TagFunc - flexible value type
//
// This function is optimized for frozen templates.
// Use ExecuteString for constantly changing templates.
func (t *Template) ExecuteString(m map[string]interface{}) (string, error) {
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
		return 0, ErrInvalidTag
	}
}

func secondIndex(s, sep []byte) int {
	n := bytes.Index(s, sep)
	if n < 0 {
		return -1 // not found
	}

	s = s[n+1:]
	lenPrev := len(s[:n]) + 1
	nSecond := bytes.Index(s, sep)
	if nSecond < 0 {
		return -1 // not found
	}

	return nSecond + lenPrev
}

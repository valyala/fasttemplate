package fasttemplate

import "errors"

var (
	ErrEmptyStartTag = errors.New("startTag cannot be empty")
	ErrEmptyEndTag = errors.New("endTag cannot be empty")
	ErrInvalidTag = errors.New("tag contains unexpected value type. Expected []byte, string or TagFunc")
)

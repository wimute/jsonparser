package jsonparser

import (
	"bytes"
	"io"
)

const (
	buffersize int = 16 * 1024
)

type BufferedReader struct {
	Bytes []byte
	IsEOF bool

	reader io.Reader
	buffer [buffersize]byte
	offset int
	level  int
}

func NewReaderBuffered() *BufferedReader {
	p := new(BufferedReader)
	return p
}

func (p *BufferedReader) ApplyReader(reader io.Reader) {
	p.reader = reader

	p.offset = 0
	p.level = 0
	p.IsEOF = false
	p.Bytes = p.buffer[p.offset:p.level]
}

func (p *BufferedReader) Clear() {
	p.offset = 0
	p.level = 0
	p.Bytes = p.buffer[p.offset:p.level]
}

func (p *BufferedReader) SkipBytes(n int) error {
	if p.offset+n >= p.level {
		p.Clear()

		if p.IsEOF {
			return nil
		}
		return p.Read()
	}
	p.offset = p.offset + n
	p.Bytes = p.buffer[p.offset:p.level]

	return nil
}

func (p *BufferedReader) TrimPrefix(offset int) {
	if (offset + p.offset) >= p.level {
		p.Clear()
		return
	}

	// Copy buffer
	i := 0
	for n := p.offset + offset; n < buffersize; n++ {
		p.buffer[i] = p.buffer[n]
		i++
	}

	// correct slice
	p.offset = 0
	p.level = p.level - offset
	p.Bytes = p.buffer[p.offset:p.level]
}

func (p *BufferedReader) Read() error {
	n, err := p.reader.Read(p.buffer[p.level:])
	if err != io.EOF && err != nil {
		return err
	}

	p.IsEOF = err == io.EOF
	p.level = p.level + n
	p.Bytes = p.buffer[:p.level]

	return nil
}

type StreamingParserCallback interface {
	HandleDataTypeRaw(reader io.Reader, dataType ValueType, parser StreamingParser) error
}

type StreamingParserCallbackTyped interface {
	HandleString(reader io.Reader)
	HandleNumber(reader io.Reader)
	HandleBoolean(value bool)
	HandleNull()
	HandleObject(reader io.Reader)
	HandleObjectPropertyKey(key io.Reader)
	HandleObjectPropertyValue()
	HandleArray(reader io.Reader)
	HandleArrayItem()
}

type StreamingParser struct {
	Callback StreamingParserCallback
}

func (p StreamingParser) Parse(data *BufferedReader) error {
	// Go to closest value
	err := skipWitespaces(data)
	if err == io.EOF {
		return MalformedJsonError
	} else if err != nil {
		return err
	}

	dataType, err := getTypeStream(data)
	if err != nil {
		return err
	}

	return p.Callback.HandleDataTypeRaw(data, dataType, p)
}

func (p StreamingParser) callCallback(data *BufferedReader, dataType ValueType) error {
	switch dataType {
	case String:

	case Number:
	case Boolean:
	case Null:
	case Object:
	case Array:
	case Unknown:
	case NotExist:
	}

	return nil
}

// Find position of next character which is not whitespace
func skipWitespaces(data *BufferedReader) error {
	for i, c := range data.Bytes {
		switch c {
		case ' ', '\n', '\r', '\t':
			continue
		default:

			return nil
		}
	}

	return nil
}

func getTypeStream(data *BufferedReader) (ValueType, error) {
	var dataType ValueType
	endOffset := offset

	// if string value
	if data[offset] == '"' {
		dataType = String
		if idx, _ := stringEnd(data[offset+1:]); idx != -1 {
			endOffset += idx + 1
		} else {
			return nil, dataType, offset, MalformedStringError
		}
	} else if data[offset] == '[' { // if array value
		dataType = Array
		// break label, for stopping nested loops
		endOffset = blockEnd(data[offset:], '[', ']')

		if endOffset == -1 {
			return nil, dataType, offset, MalformedArrayError
		}

		endOffset += offset
	} else if data[offset] == '{' { // if object value
		dataType = Object
		// break label, for stopping nested loops
		endOffset = blockEnd(data[offset:], '{', '}')

		if endOffset == -1 {
			return nil, dataType, offset, MalformedObjectError
		}

		endOffset += offset
	} else {
		// Number, Boolean or None
		end := tokenEnd(data[endOffset:])

		if end == -1 {
			return nil, dataType, offset, MalformedValueError
		}

		value := data[offset : endOffset+end]

		switch data[offset] {
		case 't', 'f': // true or false
			if bytes.Equal(value, trueLiteral) || bytes.Equal(value, falseLiteral) {
				dataType = Boolean
			} else {
				return nil, Unknown, offset, UnknownValueTypeError
			}
		case 'u', 'n': // undefined or null
			if bytes.Equal(value, nullLiteral) {
				dataType = Null
			} else {
				return nil, Unknown, offset, UnknownValueTypeError
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
			dataType = Number
		default:
			return nil, Unknown, offset, UnknownValueTypeError
		}

		endOffset += end
	}
	return data[offset:endOffset], dataType, endOffset, nil
}

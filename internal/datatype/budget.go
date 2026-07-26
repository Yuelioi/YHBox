package datatype

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
)

var typeVersionPattern = regexp.MustCompile(`^v[1-9][0-9]*$`)

func inspectJSONBudget(raw []byte, maxDepth, maxNodes int) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	depth, nodes := 0, 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			if depth != 0 {
				return errors.New("unbalanced JSON containers")
			}
			return nil
		}
		if err != nil {
			return err
		}
		nodes++
		if nodes > maxNodes {
			return errors.New("JSON node budget exceeded")
		}
		switch value := token.(type) {
		case json.Delim:
			switch value {
			case '{', '[':
				depth++
				if depth > maxDepth {
					return errors.New("JSON depth budget exceeded")
				}
			case '}', ']':
				depth--
			}
		case string:
			if len(value) > MaxSchemaResourceBytes {
				return errors.New("JSON string budget exceeded")
			}
		}
	}
}

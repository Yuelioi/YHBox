package macro

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	MediaType            = "application/vnd.yotta.macro+json"
	MaxEncodedMacroBytes = 4 << 20
)

func Encode(writer io.Writer, document Document) error {
	if writer == nil {
		return errors.New("macro encoder requires a destination")
	}
	if err := Validate(document); err != nil {
		return err
	}
	content, err := artifact.Marshal(document)
	if err != nil {
		return fmt.Errorf("encode macro: %w", err)
	}
	if len(content) > MaxEncodedMacroBytes {
		return errors.New("macro exceeds byte budget")
	}
	_, err = writer.Write(content)
	return err
}

func Decode(reader io.Reader) (Document, error) {
	if reader == nil {
		return Document{}, errors.New("macro decoder requires a source")
	}
	content, err := io.ReadAll(io.LimitReader(reader, MaxEncodedMacroBytes+1))
	if err != nil {
		return Document{}, fmt.Errorf("read macro: %w", err)
	}
	if len(content) == 0 || len(content) > MaxEncodedMacroBytes {
		return Document{}, errors.New("macro carrier is invalid")
	}
	canonical, err := artifact.Canonicalize(content)
	if err != nil || !bytes.Equal(canonical, content) {
		return Document{}, errors.New("macro carrier is not canonical")
	}
	var document Document
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return Document{}, fmt.Errorf("decode macro: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Document{}, errors.New("macro carrier contains trailing JSON")
	}
	if err := Validate(document); err != nil {
		return Document{}, err
	}
	return document, nil
}

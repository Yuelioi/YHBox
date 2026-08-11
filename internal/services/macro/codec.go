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
	decoded, err := decode(reader)
	return decoded.Document, err
}

type decodedDocument struct {
	Document      Document
	SourceVersion int
}

type documentV1 struct {
	SchemaVersion  int        `json:"schemaVersion"`
	BaseResolution [2]int     `json:"baseResolution"`
	Actions        []actionV1 `json:"actions"`
}

type actionV1 struct {
	ID         string     `json:"id"`
	Kind       ActionKind `json:"kind"`
	Key        string     `json:"key,omitempty"`
	Button     string     `json:"button,omitempty"`
	Point      *Point     `json:"point,omitempty"`
	Notches    int32      `json:"notches,omitempty"`
	DurationUs uint64     `json:"durationUs,omitempty"`
}

func decode(reader io.Reader) (decodedDocument, error) {
	if reader == nil {
		return decodedDocument{}, errors.New("macro decoder requires a source")
	}
	content, err := io.ReadAll(io.LimitReader(reader, MaxEncodedMacroBytes+1))
	if err != nil {
		return decodedDocument{}, fmt.Errorf("read macro: %w", err)
	}
	if len(content) == 0 || len(content) > MaxEncodedMacroBytes {
		return decodedDocument{}, errors.New("macro carrier is invalid")
	}
	canonical, err := artifact.Canonicalize(content)
	if err != nil || !bytes.Equal(canonical, content) {
		return decodedDocument{}, errors.New("macro carrier is not canonical")
	}
	var header struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(content, &header); err != nil {
		return decodedDocument{}, fmt.Errorf("decode macro header: %w", err)
	}
	var document Document
	switch header.SchemaVersion {
	case 1:
		var legacy documentV1
		if err := decodeStrict(content, &legacy); err != nil {
			return decodedDocument{}, err
		}
		document = migrateDocumentV1(legacy)
	case SchemaVersion:
		if err := decodeStrict(content, &document); err != nil {
			return decodedDocument{}, err
		}
	default:
		return decodedDocument{}, fmt.Errorf("macro schemaVersion %d is unsupported", header.SchemaVersion)
	}
	if err := Validate(document); err != nil {
		return decodedDocument{}, err
	}
	return decodedDocument{Document: document, SourceVersion: header.SchemaVersion}, nil
}

func decodeStrict(content []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode macro: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("macro carrier contains trailing JSON")
	}
	return nil
}

func migrateDocumentV1(legacy documentV1) Document {
	actions := make([]Action, len(legacy.Actions))
	for index, action := range legacy.Actions {
		actions[index] = Action{
			ID: action.ID, Kind: action.Kind, Key: action.Key, Button: action.Button,
			Point: clonePoint(action.Point), Notches: action.Notches, DurationUs: action.DurationUs,
		}
	}
	return Document{
		SchemaVersion:  SchemaVersion,
		BaseResolution: legacy.BaseResolution,
		// V1 dispatched pointer actions directly at their stored point. Keep
		// automatic travel disabled so migration does not add a new 300 ms move.
		Meta:    legacyV1Meta(),
		Actions: actions,
	}
}

func legacyV1Meta() Meta {
	return Meta{AutoMove: AutoMove{Enabled: false, Mode: "instant", DurationMilliseconds: 0}}
}

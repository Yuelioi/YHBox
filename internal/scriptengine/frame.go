package scriptengine

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/yottaapp/yotta/internal/artifact"
)

func WriteRequest(writer io.Writer, request Request) error {
	if err := request.Validate(); err != nil {
		return err
	}
	return writeFrame(writer, request)
}

func ReadRequest(reader io.Reader) (Request, error) {
	var request Request
	if err := readFrame(reader, &request); err != nil {
		return Request{}, err
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

func WriteResponse(writer io.Writer, response Response) error {
	if err := response.Validate(); err != nil {
		return err
	}
	return writeFrame(writer, response)
}

func ReadResponse(reader io.Reader) (Response, error) {
	var response Response
	if err := readFrame(reader, &response); err != nil {
		return Response{}, err
	}
	if err := response.Validate(); err != nil {
		return Response{}, err
	}
	return response, nil
}

func writeFrame(writer io.Writer, value any) error {
	payload, err := artifact.Marshal(value)
	if err != nil {
		return err
	}
	if len(payload) == 0 || len(payload) > MaxFrameBytes {
		return fmt.Errorf("script worker frame exceeds %d bytes", MaxFrameBytes)
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if err := writeAll(writer, header[:]); err != nil {
		return fmt.Errorf("write script worker frame header: %w", err)
	}
	if err := writeAll(writer, payload); err != nil {
		return fmt.Errorf("write script worker frame payload: %w", err)
	}
	return nil
}

func readFrame(reader io.Reader, target any) error {
	var header [4]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return fmt.Errorf("read script worker frame header: %w", err)
	}
	size := int(binary.BigEndian.Uint32(header[:]))
	if size <= 0 || size > MaxFrameBytes {
		return fmt.Errorf("script worker frame length must be within 1..%d", MaxFrameBytes)
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return fmt.Errorf("read script worker frame payload: %w", err)
	}
	var trailing [1]byte
	n, err := reader.Read(trailing[:])
	if n != 0 || !errors.Is(err, io.EOF) {
		return errors.New("script worker stream contains trailing data")
	}
	canonical, err := artifact.Canonicalize(payload)
	if err != nil {
		return fmt.Errorf("canonicalize script worker frame: %w", err)
	}
	if !bytes.Equal(canonical, payload) {
		return errors.New("script worker frame is not RFC 8785 canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode script worker frame: %w", err)
	}
	if decoder.More() {
		return errors.New("script worker frame contains multiple JSON values")
	}
	return nil
}

func writeAll(writer io.Writer, value []byte) error {
	for len(value) > 0 {
		written, err := writer.Write(value)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		value = value[written:]
	}
	return nil
}

package report

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
)

// LinkPrefix starts the URL fragment that carries a report.
const LinkPrefix = "#r="

// Link returns a link to the report page at base carrying s in its fragment, which browsers never send to a server.
func Link(base string, s Snapshot) (string, error) {
	enc, err := Encode(s)
	return base + LinkPrefix + enc, err
}

// Encode returns s as unpadded base64url of gzipped JSON.
func Encode(s Snapshot) (string, error) {
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err := json.NewEncoder(zw).Encode(s); err != nil {
		return "", err
	}
	if err := zw.Close(); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

// Decode reverses Encode.
func Decode(text string) (Snapshot, error) {
	var s Snapshot
	raw, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return s, err
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return s, err
	}
	data, err := io.ReadAll(io.LimitReader(zr, 1<<20))
	if err != nil {
		return s, err
	}
	return s, json.Unmarshal(data, &s)
}

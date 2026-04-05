package gonest

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
)

// StreamableFile wraps an io.Reader for streaming file download responses.
// It sets appropriate headers (Content-Type, Content-Disposition, Content-Length)
// automatically. Equivalent to NestJS StreamableFile.
//
// Usage in a handler:
//
//	func (c *DownloadController) Register(r gonest.Router) {
//	    r.Get("/download", func(ctx gonest.Context) error {
//	        file, _ := os.Open("report.pdf")
//	        sf := gonest.NewStreamableFile(file,
//	            gonest.WithFileName("report.pdf"),
//	            gonest.WithContentType("application/pdf"),
//	        )
//	        return sf.Send(ctx)
//	    })
//	}
type StreamableFile struct {
	reader      io.Reader
	contentType string
	disposition string
	length      int64
	fileName    string
}

// StreamableFileOption configures a StreamableFile.
type StreamableFileOption func(*StreamableFile)

// WithContentType sets the Content-Type header.
func WithContentType(ct string) StreamableFileOption {
	return func(sf *StreamableFile) {
		sf.contentType = ct
	}
}

// WithFileName sets the download file name and Content-Disposition to attachment.
func WithFileName(name string) StreamableFileOption {
	return func(sf *StreamableFile) {
		sf.fileName = name
		sf.disposition = fmt.Sprintf(`attachment; filename="%s"`, name)
	}
}

// WithDisposition sets the Content-Disposition header directly.
func WithDisposition(disposition string) StreamableFileOption {
	return func(sf *StreamableFile) {
		sf.disposition = disposition
	}
}

// WithLength sets the Content-Length header.
func WithLength(length int64) StreamableFileOption {
	return func(sf *StreamableFile) {
		sf.length = length
	}
}

// NewStreamableFile creates a new StreamableFile from a reader.
func NewStreamableFile(reader io.Reader, opts ...StreamableFileOption) *StreamableFile {
	sf := &StreamableFile{
		reader:      reader,
		contentType: "application/octet-stream",
	}
	for _, opt := range opts {
		opt(sf)
	}
	// Auto-detect content type from file name if not explicitly set
	if sf.fileName != "" && sf.contentType == "application/octet-stream" {
		if ct := mime.TypeByExtension(filepath.Ext(sf.fileName)); ct != "" {
			sf.contentType = ct
		}
	}
	return sf
}

// NewStreamableFileFromBytes creates a StreamableFile from a byte slice.
func NewStreamableFileFromBytes(data []byte, opts ...StreamableFileOption) *StreamableFile {
	opts = append([]StreamableFileOption{WithLength(int64(len(data)))}, opts...)
	return NewStreamableFile(&bytesReader{data: data, pos: 0}, opts...)
}

type bytesReader struct {
	data []byte
	pos  int
}

func (br *bytesReader) Read(p []byte) (int, error) {
	if br.pos >= len(br.data) {
		return 0, io.EOF
	}
	n := copy(p, br.data[br.pos:])
	br.pos += n
	return n, nil
}

// Send streams the file to the HTTP response.
func (sf *StreamableFile) Send(ctx Context) error {
	w := ctx.ResponseWriter()
	w.Header().Set("Content-Type", sf.contentType)

	if sf.disposition != "" {
		w.Header().Set("Content-Disposition", sf.disposition)
	}
	if sf.length > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(sf.length, 10))
	}

	w.WriteHeader(http.StatusOK)

	if closer, ok := sf.reader.(io.Closer); ok {
		defer closer.Close()
	}
	_, err := io.Copy(w, sf.reader)
	return err
}

// GetContentType returns the configured content type.
func (sf *StreamableFile) GetContentType() string { return sf.contentType }

// GetDisposition returns the configured content disposition.
func (sf *StreamableFile) GetDisposition() string { return sf.disposition }

// GetLength returns the configured content length.
func (sf *StreamableFile) GetLength() int64 { return sf.length }

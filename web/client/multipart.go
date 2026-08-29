package client

import (
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"sort"
	"strings"
	"sync"
)

const defaultMultipartFileContentType = "application/octet-stream"

// MultipartFile 描述 multipart/form-data 请求中的文件字段。
type MultipartFile struct {
	// FieldName 是 multipart 表单字段名。
	FieldName string
	// FileName 是客户端提交的文件名。
	FileName string
	// ContentType 是文件段媒体类型，空值使用 application/octet-stream。
	ContentType string
	// Body 是文件内容流，nil 表示空文件段。
	Body io.Reader
}

// WithMultipartFields 写入 multipart/form-data 请求体，普通字段只保留单值。
func WithMultipartFields(fields map[string]string, files ...MultipartFile) RequestOption {
	values := make(url.Values, len(fields))
	for name, value := range fields {
		values.Set(name, value)
	}
	return WithMultipartBody(values, files...)
}

// WithMultipartBody 写入 multipart/form-data 请求体，支持多值普通字段和文件字段。
func WithMultipartBody(fields url.Values, files ...MultipartFile) RequestOption {
	copiedFields := cloneURLValues(fields)
	copiedFiles := cloneMultipartFiles(files)
	err := validateMultipartConfig(copiedFields, copiedFiles)
	return func(config *requestConfig) error {
		if err != nil {
			return err
		}
		body, contentType := newMultipartBody(copiedFields, copiedFiles)
		config.body = body
		if config.headers.Get("Content-Type") == "" {
			config.headers.Set("Content-Type", contentType)
		}
		return nil
	}
}

type multipartBody struct {
	once   sync.Once
	reader *io.PipeReader
	writer *io.PipeWriter
	form   *multipart.Writer
	fields url.Values
	files  []MultipartFile
	done   chan struct{}

	mu  sync.Mutex
	err error
}

func newMultipartBody(fields url.Values, files []MultipartFile) (*multipartBody, string) {
	reader, writer := io.Pipe()
	form := multipart.NewWriter(writer)
	body := &multipartBody{
		reader: reader,
		writer: writer,
		form:   form,
		fields: cloneURLValues(fields),
		files:  cloneMultipartFiles(files),
		done:   make(chan struct{}),
	}
	return body, form.FormDataContentType()
}

func (b *multipartBody) Read(p []byte) (int, error) {
	b.once.Do(b.start)
	return b.reader.Read(p)
}

func (b *multipartBody) Close() error {
	b.once.Do(b.start)
	err := b.reader.Close()
	<-b.done
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		return err
	}
	writeErr := b.writeError()
	if errors.Is(writeErr, io.ErrClosedPipe) {
		return nil
	}
	return writeErr
}

func (b *multipartBody) start() {
	go func() {
		err := writeMultipart(b.form, b.fields, b.files)
		closeErr := b.form.Close()
		if err == nil {
			err = closeErr
		}
		b.setWriteError(err)
		_ = b.writer.CloseWithError(err)
		close(b.done)
	}()
}

func (b *multipartBody) setWriteError(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.err = err
}

func (b *multipartBody) writeError() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.err
}

func writeMultipart(writer *multipart.Writer, fields url.Values, files []MultipartFile) error {
	for _, name := range sortedValueKeys(fields) {
		for _, value := range fields[name] {
			if err := writer.WriteField(name, value); err != nil {
				return err
			}
		}
	}
	for _, file := range files {
		if err := writeMultipartFile(writer, file); err != nil {
			return err
		}
	}
	return nil
}

func writeMultipartFile(writer *multipart.Writer, file MultipartFile) error {
	header := make(textproto.MIMEHeader)
	disposition := map[string]string{"name": file.FieldName}
	if file.FileName != "" {
		disposition["filename"] = file.FileName
	}
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", disposition))
	contentType := strings.TrimSpace(file.ContentType)
	if contentType == "" {
		contentType = defaultMultipartFileContentType
	}
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	if file.Body == nil {
		return nil
	}
	_, err = io.Copy(part, file.Body)
	return err
}

func validateMultipartConfig(fields url.Values, files []MultipartFile) error {
	for name := range fields {
		if strings.TrimSpace(name) == "" {
			return ErrInvalidRequest
		}
	}
	for _, file := range files {
		if strings.TrimSpace(file.FieldName) == "" {
			return ErrInvalidRequest
		}
	}
	return nil
}

func sortedValueKeys(values url.Values) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cloneURLValues(values url.Values) url.Values {
	if len(values) == 0 {
		return nil
	}
	cloned := make(url.Values, len(values))
	for name, list := range values {
		cloned[name] = append([]string(nil), list...)
	}
	return cloned
}

func cloneMultipartFiles(files []MultipartFile) []MultipartFile {
	if len(files) == 0 {
		return nil
	}
	cloned := make([]MultipartFile, len(files))
	copy(cloned, files)
	return cloned
}

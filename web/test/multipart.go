package webtest

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
)

const defaultMultipartFileContentType = "application/octet-stream"

// MultipartFile 描述测试请求中的 multipart 文件字段。
type MultipartFile struct {
	FieldName   string
	FileName    string
	ContentType string
	Body        io.Reader
}

// WithMultipartBody 构造 multipart/form-data 请求体。
func WithMultipartBody(fields map[string]string, files ...MultipartFile) RequestOption {
	return func(config *requestConfig) error {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for name, value := range fields {
			if err := writer.WriteField(name, value); err != nil {
				return err
			}
		}
		for _, file := range files {
			if err := writeMultipartFile(writer, file); err != nil {
				return err
			}
		}
		if err := writer.Close(); err != nil {
			return err
		}
		config.body = bytes.NewReader(body.Bytes())
		config.headers.Set("Content-Type", writer.FormDataContentType())
		return nil
	}
}

func writeMultipartFile(writer *multipart.Writer, file MultipartFile) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name":     file.FieldName,
		"filename": file.FileName,
	}))
	contentType := file.ContentType
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

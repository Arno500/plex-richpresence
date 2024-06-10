package discord

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"path"
)

func UploadImage(image []byte, imageName string) (string, error) {

	reader, mpForm := createMultipartForm("", image, imageName)
	defer reader.Close()

	req, err := http.NewRequest(http.MethodPost, "https://litterbox.catbox.moe/resources/internals/api.php", reader)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", mpForm.FormDataContentType())

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	return string(body), nil
}

func createMultipartForm(expiration string, image []byte, imageName string) (*io.PipeReader, *multipart.Writer) {
	r, w := io.Pipe()
	m := multipart.NewWriter(w)

	go func() {
		defer w.Close()
		defer m.Close()

		m.WriteField("reqtype", "fileupload")
		
		if expiration != "" {
			m.WriteField("time", expiration)
		} else {
			m.WriteField("time", "12h")
		}

		formFile, err := m.CreateFormFile("fileToUpload", path.Base(imageName))
		if err != nil {
			return
		}
		if _, err := io.Copy(formFile, bytes.NewBuffer(image)); err != nil {
			return
		}
	}()

	return r, m
}
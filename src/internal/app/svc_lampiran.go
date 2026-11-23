package app

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

const lampiranMaxBytes = 2 * 1024 * 1024

var lampiranTypes = map[string]string{"application/pdf": "pdf", "image/jpeg": "jpg", "image/png": "png"}

var lampiranCtrl = regexp.MustCompile(`[\x00-\x1F\x7F]`)

// LampiranSimpan menyimpan unggahan nota/dokumen; mengembalikan path dan nama_asli.
func LampiranSimpan(fh *multipart.FileHeader) (path, namaAsli string, err error) {
	if fh == nil {
		return "", "", errors.New("Pilih berkas nota atau dokumen lebih dulu.")
	}
	if fh.Size > lampiranMaxBytes {
		return "", "", errors.New("Ukuran berkas maksimal 2 MB.")
	}
	f, e := fh.Open()
	if e != nil {
		return "", "", errors.New("Berkas unggahan tidak valid.")
	}
	defer f.Close()
	data, e := io.ReadAll(f)
	if e != nil {
		return "", "", errors.New("Berkas unggahan tidak valid.")
	}
	sniff := data
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	mime := http.DetectContentType(sniff)
	if i := indexByte(mime, ';'); i >= 0 {
		mime = mime[:i]
	}
	ext, ok := lampiranTypes[mime]
	if !ok {
		return "", "", errors.New("Gunakan berkas PDF, JPG, atau PNG.")
	}
	original := lampiranCtrl.ReplaceAllString(filepath.Base(fh.Filename), "")
	if fh.Filename == "" {
		original = "dokumen"
	}
	if original == "" {
		original = "dokumen"
	}
	if len(original) > 180 {
		original = original[:180]
	}
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	name := "lampiran_" + hex.EncodeToString(b) + "." + ext
	if os.WriteFile(UploadPath(name), data, 0o644) != nil {
		return "", "", errors.New("Gagal menyimpan berkas unggahan.")
	}
	return name, original, nil
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

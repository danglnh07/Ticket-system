package util

import "github.com/skip2/go-qrcode"

func GenerateQRCode(content, dest string) error {
	return qrcode.WriteFile(content, qrcode.Medium, 256, dest)
}

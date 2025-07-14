package converter

import "encoding/base64"

func encodeToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
package dkim

import (
	"bytes"
	"strings"
)

// ContainsInt returns true if an int is present in a iteratee.
func ContainsInt(s []int, v int) bool {
	for _, vv := range s {
		if vv == v {
			return true
		}
	}
	return false
}

func CanonicalizeHeadersSimple(headers *[]HeaderTuple, signedHeaders *[]string) string {
	canonicalizedHeaders := ""
	alreadyUsed := make([]int, 0)

	for _, signedHeader := range *signedHeaders {
		for idx, header := range *headers {
			if !ContainsInt(alreadyUsed, idx) {
				if strings.ToLower(header.one) == strings.ToLower(signedHeader) {
					canonicalizedHeaders += strings.ToLower(signedHeader)
					canonicalizedHeaders += header.tow
					canonicalizedHeaders += strings.ReplaceAll(header.three, "\r\n", "")
					canonicalizedHeaders += "\r\n"

					alreadyUsed = append(alreadyUsed, idx)
					break
				}
			}
		}
	}

	return canonicalizedHeaders
}

func CanonicalizeHeaderRelaxed(headers *[]HeaderTuple, signedHeaders *[]string) string {
	canonicalizedHeaders := ""
	alreadyUsed := make([]int, 0)

	for _, signedHeader := range *signedHeaders {
		for idx, header := range *headers {
			if !ContainsInt(alreadyUsed, idx) {
				if strings.ToLower(header.one) == strings.ToLower(signedHeader) {
					canonicalizedHeaders += strings.ToLower(signedHeader)
					canonicalizedHeaders += ":"
					canonicalizedHeaders += canonicalizeHeaderRelaxed(header.three)
					canonicalizedHeaders += "\r\n"

					alreadyUsed = append(alreadyUsed, idx)
					break
				}
			}
		}
	}

	return canonicalizedHeaders
}

func canonicalizeHeaderRelaxed(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.TrimSpace(value)
	value = trimSpace(value)
	return value
}

func trimSpace(value string) string {
	var buffer bytes.Buffer
	previous := false
	for _, c := range value {
		if c != ' ' {
			previous = false
			buffer.WriteByte(byte(c))
		} else {
			if !previous {
				previous = true
				buffer.WriteByte(byte(c))
			}
		}
	}
	return buffer.String()
}

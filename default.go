package goakeneo

import "time"

const (
	defaultHTTPTimeout = 10 * time.Second
	defaultAccept      = "application/json"
	defaultContentType = "application/json"
	defaultUserAgent   = "go-akeneo v1.0.0"
	defaultRateLimit   = 5 // 5 requests per second
	defaultVersion     = AkeneoPimVersion6
)

const (
	// AkeneoPimVersion7 is the version 7 of Akeneo PIM
	AkeneoPimVersion7 = iota + 1
	// AkeneoPimVersion6 is the version 6 of Akeneo PIM
	AkeneoPimVersion6
	// AkeneoPimVersion5 is the version 5 of Akeneo PIM
	AkeneoPimVersion5
)

var (
	pimVersionMap = map[int]string{
		AkeneoPimVersion7: "7.0",
		AkeneoPimVersion6: "6.0",
		AkeneoPimVersion5: "5.0",
	}
)
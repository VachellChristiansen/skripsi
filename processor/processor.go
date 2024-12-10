package processor

import "skripsi/helper"

type Processor struct {
	WebProcessor     WebProcessor
	WebViewProcessor WebViewProcessor
}

func NewProcessor(l helper.LoggerHelper) Processor {
	return Processor{
		WebProcessor:     NewWebProcessor(l),
		WebViewProcessor: NewWebViewProcessor(l),
	}
}

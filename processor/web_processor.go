package processor

type WebProcessor interface {

}

type WebProcessorImpl struct {

}

func NewWebProcessor() WebProcessor {
	return &WebProcessorImpl{}
}

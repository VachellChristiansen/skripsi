package processor

type IndexData struct {
	Data       map[string]interface{}
	Message    string
	Err        string
	StatusCode int
	Timestamp  int64
}

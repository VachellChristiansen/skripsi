package processor

type IndexData struct {
	Data       map[string]interface{}
	JSData     string
	Message    string
	Err        string
	StatusCode int
	Timestamp  int64
}

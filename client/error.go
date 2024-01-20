package client

import "encoding/json"

type ErrServer struct {
	Code    int
	Message string
}

func (es *ErrServer) UnmarshalJSON(data []byte) error {
	var errWeb struct {
		Code    int
		Message string
	}

	err := json.Unmarshal(data, &errWeb)
	if err != nil {
		return err
	}

	es.Code = errWeb.Code
	es.Message = errWeb.Message
	return nil
}

func (es *ErrServer) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Code    int
		Message string
	}{
		Code:    es.Code,
		Message: es.Message,
	})
}

func (es *ErrServer) Error() string {
	return es.Message
}

const ErrNoDuplicateUploads = 10006

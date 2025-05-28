package services

import "encoding/json"

type ModelSerializer interface {
	Encode(model interface{}) ([]byte, error)
	Decode(data []byte, model interface{}) error
}

type JsonModelSerializer struct{}

func NewJsonModelSerializer() *JsonModelSerializer {
	return &JsonModelSerializer{}
}

func (j *JsonModelSerializer) Encode(model interface{}) ([]byte, error) {
	data, err := json.Marshal(model)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (j *JsonModelSerializer) Decode(data []byte, model interface{}) error {
	return json.Unmarshal(data, model)
}

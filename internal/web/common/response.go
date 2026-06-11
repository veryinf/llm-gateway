package common

import "encoding/json"

type ResponseStruct struct {
	ErrCode int    `json:"errCode"`
	ErrMsg  string `json:"errMsg"`
}

func (r ResponseStruct) Error() string {
	j, _ := json.Marshal(r)
	return string(j)
}

type Data[T any] struct {
	ResponseStruct
	Data T `json:"data"`
}

type DataSet[T any] struct {
	ResponseStruct
	DataSet []T   `json:"dataSet"`
	Total   int64 `json:"total,omitempty"`
	HasMore bool  `json:"hasMore,omitempty"`
}

func NewResponse(code int, msg string) *ResponseStruct {
	return &ResponseStruct{
		ErrCode: code,
		ErrMsg:  msg,
	}
}

func NewData[T any](data T) *Data[T] {
	return &Data[T]{
		ResponseStruct: *NewResponse(0, "ok"),
		Data:           data,
	}
}

func NewDataSet[T any](dataSet []T, total int64) *DataSet[T] {
	return &DataSet[T]{
		ResponseStruct: *NewResponse(0, "ok"),
		DataSet:        dataSet,
		Total:          total,
		HasMore:        false,
	}
}

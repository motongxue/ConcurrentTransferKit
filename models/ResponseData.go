package models

type ResponseData struct {
	Code int              `json:"code"`
	Msg  string           `json:"msg"`
	Data FileTransferInfo `json:"data"`
}

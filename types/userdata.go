package types

import (
	"encoding/json"
	"fmt"
)

type RequestData struct {
	Url           string `db:"url,omitempty" json:"url,omitempty"`
	Method        string `db:"method,omitempty" json:"method,omitempty"`
	HasCookies    bool   `db:"has_cookies,omitempty" json:"has_cookies,omitempty"`
	HasParams     bool   `db:"has_params,omitempty" json:"has_params,omitempty"`
	ContentLength int    `db:"content_length,omitempty" json:"content_length,omitempty"`
	IsHTTPS       bool   `db:"is_https,omitempty" json:"is_https,omitempty"`
	Path          string `db:"path,omitempty" json:"path,omitempty"`
	Query         string `db:"query,omitempty" json:"query,omitempty"`
	Fragment      string `db:"fragment,omitempty" json:"fragment,omitempty"`
}

type ResponseData struct {
	Title         string `db:"title,omitempty" json:"title,omitempty"`
	Mimetype      string `db:"mimetype,omitempty" json:"mimetype,omitempty"`
	StatusCode    int    `db:"status_code,omitempty" json:"status_code,omitempty"`
	ContentLength int    `db:"content_length,omitempty" json:"content_length,omitempty"`
	HasCookies    bool   `db:"has_cookies,omitempty" json:"has_cookies,omitempty"`
	Date          string `db:"date,omitempty" json:"date,omitempty"`
	Time          string `db:"time,omitempty" json:"time,omitempty"`
}

type UserData struct {
	ID               string       `db:"id,omitempty" json:"id,omitempty"`
	Host             string       `db:"host,omitempty" json:"host,omitempty"`
	IP               string       `db:"ip,omitempty" json:"ip,omitempty"`
	Port             string       `db:"port,omitempty" json:"port,omitempty"`
	HasResponse      bool         `db:"has_response,omitempty" json:"has_response,omitempty"`
	IsRequestEdited  bool         `db:"is_request_edited,omitempty" json:"is_request_edited,omitempty"`
	IsResponseEdited bool         `db:"is_response_edited,omitempty" json:"is_response_edited,omitempty"`
	OriginalRequest  RequestData  `db:"original_request,omitempty" json:"original_request,omitempty"`
	OriginalResponse ResponseData `db:"original_response,omitempty" json:"original_response,omitempty"`
	EditedRequest    RequestData  `db:"edited_request,omitempty" json:"edited_request,omitempty"`
	EditedResponse   ResponseData `db:"edited_response,omitempty" json:"edited_response,omitempty"`
	StoreID          string       `db:"store_id,omitempty" json:"store_id,omitempty"`
	// Labels           []string     `db:"labels,omitempty" json:"labels,omitempty"`
}

type RealtimeRecord struct {
	CollectionId     string      `db:"collectionId" json:"collectionId"`
	CollectionName   string      `db:"collectionName" json:"collectionName"`
	Created          string      `db:"created" json:"created"`
	Updated          string      `db:"updated" json:"updated"`
	ID               string      `db:"id" json:"id"`
	Host             string      `db:"host" json:"host"`
	IP               string      `db:"ip" json:"ip"`
	Port             string      `db:"port" json:"port"`
	UrlData          interface{} `db:"url_data" json:"url_data"`
	OriginalRequest  interface{} `db:"original_request" json:"original_request"`
	OriginalResponse interface{} `db:"original_response" json:"original_response"`
	HasResponse      bool        `db:"has_response" json:"has_response"`
	IsRequestEdited  bool        `db:"is_request_edited" json:"is_request_edited"`
	IsResponseEdited bool        `db:"is_response_edited" json:"is_response_edited"`
	EditedRequest    interface{} `db:"edited_request" json:"edited_request"`
	StoreID          string      `db:"store_id,omitempty" json:"store_id,omitempty"`
	EditedResponse   interface{} `db:"edited_response" json:"edited_response"`
	// Labels           interface{} `db:"labels,omitempty" json:"labels,omitempty"`
	Action string `db:"action,omitempty" json:"action,omitempty"`
}

type OutputData struct {
	Userdata UserData
	Host     string
	Port     string
	Folder   string
}

func (d *UserData) Scan(value interface{}) error {
	if value == nil {
		*d = UserData{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, d)
	case string:
		return json.Unmarshal([]byte(v), d)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

func (d *RequestData) Scan(value interface{}) error {
	if value == nil {
		*d = RequestData{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, d)
	case string:
		return json.Unmarshal([]byte(v), d)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

func (d *ResponseData) Scan(value interface{}) error {
	if value == nil {
		*d = ResponseData{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, d)
	case string:
		return json.Unmarshal([]byte(v), d)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

// {
//     "collectionId": "0rlm6bo4w4ldxzw",
//     "collectionName": "intercept",
//     "created": "2023-03-29 12:24:16.192Z",
//     "edited_request": {},
//     "edited_response": {},
//     "has_response": true,
//     "host": "test2",
//     "id": "kVGuQP8HqUJITn1",
//     "ip": "test3",
//     "is_request_edited": true,
//     "is_response_edited": true,
//     "labels": {},
//     "original_request": {},
//     "original_response": {},
//     "port": "test3",
//     "updated": "2023-03-29 12:36:40.444Z",
//     "url_data": {}
// }

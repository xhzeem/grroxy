package api

type Endpoint struct {
	Method   string
	Endpoint string
}

// var V1 = struct {
// 	Data         Endpoint
// 	SitemapNew   Endpoint
// }{
// 	Data: Endpoint{
// 		Method:   http.MethodPost,
// 		Endpoint: "/api/data",
// 	},
// 	SitemapNew: Endpoint{
// 		Method:   http.MethodPost,
// 		Endpoint: ,
// 	}
// }

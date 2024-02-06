package detections

type Definition struct {
	Title string
	Color string
}

type Definitions struct {
	Partials   map[string]Definition `json:"partials"`
	Files      map[string]Definition `json:"files"`
	Extensions map[string]Definition `json:"extensions"`
	Default    Definition            `json:"default"`
}

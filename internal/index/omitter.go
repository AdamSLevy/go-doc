package index

import "encoding/json"

type omitter interface{ shouldOmit() bool }

func omitEmptyElementsMarshalJSON[S ~[]E, E omitter](s S) ([]byte, error) {
	list := make([]json.RawMessage, 0, len(s))
	for _, e := range s {
		if e.shouldOmit() {
			continue
		}
		data, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		list = append(list, data)
	}
	return json.Marshal(list)
}

package tools

type Params = map[string]interface{}

func AddIfNotExists(p *Params, key string, val interface{}) {
	if *p == nil {
		*p = make(map[string]interface{})
	}

	data := *p
	if _, ok := data[key]; !ok {
		data[key] = val
	}
}

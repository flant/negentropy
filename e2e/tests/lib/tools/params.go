package tools

type Params map[string]interface{}

func (p *Params) AddIfNotExists(key string, val interface{}) {
	if *p == nil {
		*p = make(map[string]interface{})
	}

	data := *p
	if _, ok := data[key]; !ok {
		data[key] = val
	}
}

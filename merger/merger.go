package merger

type Source struct {
	GroupName     string
	ObjProperties map[string]string
}

type Result = map[string]map[string]string

func MergeSync(sources []*Source) (Result, error) {
	res := make(Result)

	for _, src := range sources {

		for name, val := range src.ObjProperties {
			if _, ok := res[name]; !ok {
				res[name] = make(map[string]string)
			}
			res[name][src.GroupName] = val
		}
	}
	return res, nil
}

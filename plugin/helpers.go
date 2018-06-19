package plugin

func contains(slice []*string, item *string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		if s != nil {
			set[*s] = struct{}{}
		}
	}

	if item == nil {
		return false
	}

	_, ok := set[*item]
	return ok
}

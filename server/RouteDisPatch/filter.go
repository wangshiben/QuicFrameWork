package RouteDisPatch

import "strings"

func (r *Route) ensureRouteExists(pathSegment string) (*Route, bool) {
	index, exists := r.Index[pathSegment]
	if !exists {
		newRoute := &Route{
			path:      pathSegment,
			Filter:    []HttpFilter{},
			NextRoute: []*Route{},
			Index:     map[string]int{},
		}
		r.NextRoute = append(r.NextRoute, newRoute)
		r.Index[pathSegment] = len(r.NextRoute) - 1
		return newRoute, false
	}
	return r.NextRoute[index], true
}
func (r *Route) AddFilter(path string, filter HttpFilter) {
	format := formatPath(path)
	r.addFilter(format, filter)
}

func (r *Route) addFilter(path string, filter HttpFilter) {
	parts := strings.SplitN(path, "/", 2)
	// Handle the first segment of the path
	nextRoute, _ := r.ensureRouteExists(parts[0])
	if len(parts) == 1 {
		// Only one part, so we're at the final route to add the filter
		if nextRoute.Filter == nil {
			nextRoute.Filter = []HttpFilter{}
		}
		nextRoute.Filter = append(nextRoute.Filter, filter)
		return
	}

	// Recursive call for the remainder of the path
	nextRoute.addFilter(parts[1], filter)
}

func (r *Route) getFilter(path string, FilterChain []HttpFilter) []HttpFilter {
	index, exist := 0, false
	tempChains := make([]HttpFilter, 0)
	routes := strings.SplitN(path, "/", 2)
	if len(routes) == 1 {
		index, exist = r.Index[routes[0]]
	} else {
		index, exist = r.Index[routes[0]]
	}
	if len(routes) == 1 {
		if exist { //匹配到末尾的拦截器链
			tempChains = append(tempChains, r.NextRoute[index].Filter...)
		}
	} else {
		if exist {
			tempChains = append(tempChains, r.NextRoute[index].getFilter(routes[1], FilterChain)...)
		}
	}

	for in, item := range r.NextRoute {
		if in == index { //防止重复添加
			continue
		}
		switch item.path {
		case "*":
			FilterChain = append(FilterChain, item.Filter...)
			FilterChain = append(FilterChain, item.getFilter(routes[1], FilterChain)...)
		case "**":
			FilterChain = append(FilterChain, item.Filter...)
			return append(FilterChain, tempChains...)
		}
	}
	return append(FilterChain, tempChains...)
}

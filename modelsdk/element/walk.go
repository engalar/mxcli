package element

// Walk recursively traverses an Element tree in depth-first order.
// fn is called for each element; return false to stop traversal.
func Walk(root Element, fn func(elem Element) bool) {
	walkImpl(root, fn)
}

func walkImpl(root Element, fn func(Element) bool) bool {
	if !fn(root) {
		return false
	}
	for _, prop := range root.Properties() {
		switch p := prop.(type) {
		case ChildProperty:
			if child := p.ChildElement(); child != nil {
				if !walkImpl(child, fn) {
					return false
				}
			}
		case ChildListProperty:
			for _, child := range p.ChildElements() {
				if !walkImpl(child, fn) {
					return false
				}
			}
		}
	}
	return true
}

// FindByName searches the element tree for the first element whose
// Name() method returns the given name. Returns nil if not found.
func FindByName(root Element, name string) Element {
	type namer interface{ Name() string }
	var found Element
	Walk(root, func(e Element) bool {
		if n, ok := e.(namer); ok && n.Name() == name {
			found = e
			return false
		}
		return true
	})
	return found
}

// WalkStrings visits every WritableProperty on every element in the tree,
// calling fn for each string-valued property. Return false to stop.
func WalkStrings(root Element, fn func(elem Element, propName string, value string) bool) {
	walkStringsImpl(root, fn)
}

func walkStringsImpl(root Element, fn func(Element, string, string) bool) bool {
	for _, prop := range root.Properties() {
		if wp, ok := prop.(WritableProperty); ok {
			if v := wp.BSONValue(); v != nil {
				if s, ok := v.(string); ok && s != "" {
					if !fn(root, prop.Name(), s) {
						return false
					}
				}
			}
		}
		switch p := prop.(type) {
		case ChildProperty:
			if child := p.ChildElement(); child != nil {
				if !walkStringsImpl(child, fn) {
					return false
				}
			}
		case ChildListProperty:
			for _, child := range p.ChildElements() {
				if !walkStringsImpl(child, fn) {
					return false
				}
			}
		}
	}
	return true
}

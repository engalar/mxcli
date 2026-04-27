package property

// Enum is a property that holds a single enumeration value (a constrained string type).
type Enum[T ~string] struct {
	propertyBase
	val T
}

func NewEnum[T ~string](name string) *Enum[T] {
	return &Enum[T]{propertyBase: propertyBase{name: name}}
}

func (e *Enum[T]) Get() T         { return e.val }
func (e *Enum[T]) BSONValue() any { return string(e.val) }

// Set stores the value and marks the property dirty.
func (e *Enum[T]) Set(v T) {
	e.val = v
	e.markDirty()
}

// SetFromDecode populates the value during BSON decode without marking dirty.
func (e *Enum[T]) SetFromDecode(v T) {
	e.val = v
}

// EnumList is a property that holds an ordered list of enumeration values.
type EnumList[T ~string] struct {
	propertyBase
	items []T
}

func NewEnumList[T ~string](name string) *EnumList[T] {
	return &EnumList[T]{propertyBase: propertyBase{name: name}}
}

func (e *EnumList[T]) Items() []T { return e.items }
func (e *EnumList[T]) BSONValue() any {
	out := make([]string, len(e.items))
	for i, v := range e.items {
		out[i] = string(v)
	}
	return out
}

// Append adds a value to the list and marks the property dirty.
func (e *EnumList[T]) Append(v T) {
	e.items = append(e.items, v)
	e.markDirty()
}

// SetFromDecode replaces the list during BSON decode without marking dirty.
func (e *EnumList[T]) SetFromDecode(items []T) {
	e.items = items
}

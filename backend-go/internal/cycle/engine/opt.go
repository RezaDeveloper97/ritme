package engine

// Opt is an optional value (JSON null when absent). The zero value is "none".
type Opt[T any] struct {
	v  T
	ok bool
}

// Some is a present value.
func Some[T any](v T) Opt[T] { return Opt[T]{v: v, ok: true} }

// None is an absent value.
func None[T any]() Opt[T] { return Opt[T]{} }

// Get returns the value and whether it is present.
func (o Opt[T]) Get() (T, bool) { return o.v, o.ok }

// OK reports whether the value is present.
func (o Opt[T]) OK() bool { return o.ok }

// Or returns the value, or def when absent.
func (o Opt[T]) Or(def T) T {
	if o.ok {
		return o.v
	}
	return def
}

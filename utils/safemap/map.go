package safemap

type Map interface {
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
	Has(key string) bool
	Remove(key string)
	Count() int
	Keys() []string
}

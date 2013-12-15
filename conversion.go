package xweb

// a struct implements this interface can be convert from request param to a struct
type FromConversion interface {
    FromString(content string) error
}

// a struct implements this interface can be convert from struct to template variable
// Not Implemented
type ToConversion interface {
    ToString() string
}

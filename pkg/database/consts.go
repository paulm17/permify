package database

// Engine - Type declaration of engine
type Engine string

const (
	POSTGRES Engine = "postgres"
	YUGABYTE Engine = "yugabyte"
	MEMORY   Engine = "memory"
)

// String - Convert to string
func (c Engine) String() string {
	return string(c)
}

const (
	_defaultPageSize = 50
)

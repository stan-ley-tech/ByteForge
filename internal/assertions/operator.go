package assertions

type operator int

const (
	opEquals operator = iota
	opNotEquals
	opLessThan
	opLessOrEqual
	opGreaterThan
	opGreaterOrEqual
	opContains
	opExists
	opNotExists
)

func (o operator) symbol() string {
	switch o {
	case opEquals:
		return "=="
	case opNotEquals:
		return "!="
	case opLessThan:
		return "<"
	case opLessOrEqual:
		return "<="
	case opGreaterThan:
		return ">"
	case opGreaterOrEqual:
		return ">="
	case opContains:
		return "contains"
	case opExists:
		return "exists"
	case opNotExists:
		return "not exists"
	default:
		return "?"
	}
}

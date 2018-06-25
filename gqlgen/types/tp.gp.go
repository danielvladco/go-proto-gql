package types

type Test struct {
	Name      string
	Phone     string
	Companies []*Company
}

type Company struct {
	Name   string
	Period *Period
}

type Period struct {
	From    string
	To      string
	Current bool
}

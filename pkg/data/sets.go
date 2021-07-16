package data

type StringSet map[string]struct{}

//Items returns the strings in the set as a slice.
func (s StringSet) Items() []string {
	retVal := make([]string, 0, len(s))
	for str := range s {
		retVal = append(retVal, str)
	}
	return retVal
}

//Insert adds a string to the set
func (s StringSet) Insert(str string) {
	if s == nil {
		s = make(StringSet)
	}
	s[str] = struct{}{}
}

//Contains checks if a given string is in the set
func (s StringSet) Contains(str string) bool {
	if s == nil {
		return false
	}
	_, ok := s[str]
	return ok
}

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
	s[str] = struct{}{}
}

//Contains checks if a given string is in the set
func (s StringSet) Contains(str string) bool {
	_, ok := s[str]
	return ok
}

type IntSet map[int]struct{}

//Items returns the integers in the set as a slice.
func (s IntSet) Items() []int {
	retVal := make([]int, 0, len(s))
	for intVal := range s {
		retVal = append(retVal, intVal)
	}
	return retVal
}

//Insert adds a integer to the set
func (s IntSet) Insert(intVal int) {
	s[intVal] = struct{}{}
}

//Contains checks if a given integer is in the set
func (s IntSet) Contains(intVal int) bool {
	_, ok := s[intVal]
	return ok
}

type Int64Set map[int64]struct{}

//Items returns the integers in the set as a slice.
func (s Int64Set) Items() []int64 {
	retVal := make([]int64, 0, len(s))
	for intVal := range s {
		retVal = append(retVal, intVal)
	}
	return retVal
}

//Insert adds a integer to the set
func (s Int64Set) Insert(intVal int64) {
	s[intVal] = struct{}{}
}

//Contains checks if a given integer is in the set
func (s Int64Set) Contains(intVal int64) bool {
	_, ok := s[intVal]
	return ok
}

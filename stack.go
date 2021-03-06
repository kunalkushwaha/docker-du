package  main

type Stack struct {
	data []interface{}
}

func NewStack() *Stack {
	s := Stack{data: make([]interface{},0)} //erface{})}
	return &s
}

func (s *Stack) Push(data interface{}) {
	s.data = append(s.data, data)
}

func (s *Stack) Pop() interface{} {
	if len(s.data) == 0 {
		return nil
	}
	data := s.data[len(s.data) -1]
	s.data = s.data[0:len(s.data)-1]
	return data 
}

func (s *Stack) Peek() interface{} {
	if len(s.data) == 0 {
		return nil
	}
	return s.data[len(s.data)-1]
}

func (s *Stack) Dequeue() interface{} {
	if len(s.data) == 0{
		return nil
	}

	data := s.data[0]
	s.data = s.data[1:len(s.data)]
	return data
}

func (s *Stack) Size() int {
	return 	len(s.data)
}
/*
func (s *Stack) Dump() {
	fmt.Println("--------- Stack --------------")
	for _, elem := range s.data {
		fmt.Println(elem)
	}
	fmt.Println("--------- Stack --------------")
}
*/

package main 

import "testing"
import "fmt"

func TestPushPop(t *testing.T) {
	s := NewStack()
	s.Push(10)
	s.Push("Hello")
	s.Push([]string{"abc"," xyz"})
	fmt.Println(s.Pop())
	fmt.Println(s.Pop())
}

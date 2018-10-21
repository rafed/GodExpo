package main

import (
	"fmt"
	"strings"
)

func lol() {
	hello := "Hello"
	world := "World"
	words := []string{hello, world}
	SayHello(words)

	person := person{"Rafed", 25}
	person.Hi()
	var i = 0
	i++
}

// SayHello says Hello
func SayHello(words []string) {
	fmt.Println(joinStrings(words))
}

// joinStrings joins strings
func joinStrings(words []string) string {
	return strings.Join(words, ", ")
}

type person struct {
	name string
	age  int
}

type rectangle struct {
	length int
	height int
}

type cube struct {
	rect  rectangle
	width int
}

func (c *cube) volume(r rectangle) {
	println("volume is", 23)
}

func (p person) Hi() {
	if p.age > 40 {
		fmt.Print("Dont mention it.")
	} else {
		fmt.Print("lala")
	}
	fmt.Printf("My name is %s. I'm %d years old.\n", p.name, p.age)
}

func (p *person) Bye(name string) {
	println("Bye bye", name)
}

func lala(age int) {
	println("I dont wanna say my age")
}

type mojo struct {
}

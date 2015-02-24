package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/tiborvass/go-jsx"
)

/*
For tests
---------
s := `var props = {x: 42};
var fooName = "foo";
var style = "nostyle";
var x = <p>{hello}</p>;
var app = <Nav {...props} color="blue" style={style}>Hello {<Foo name={fooName}/>} world<a href={props.x}>link</a></Nav>;
console.log("hello");`
*/

func usage() {
	fmt.Println("jsx converts jsx files to JavaScript code calling React.js and outputs the result to stdout.")
	fmt.Println()
	fmt.Println("jsx <filename>")
	fmt.Println("\treads from the file named `filename`")
	fmt.Println("jsx -")
	fmt.Println("\treads from stdin")
	fmt.Println()
	os.Exit(0)
}

func main() {
	log.SetFlags(0)
	var (
		res string
		err error
	)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "--help":
			usage()
		case "-":
			stdin, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				log.Fatal(err)
			}
			res, err = jsx.String(string(stdin))
		default:
			res, err = jsx.File(os.Args[1])
		}
	} else {
		usage()
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
}

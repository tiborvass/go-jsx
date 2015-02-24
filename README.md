# React JSX without Node.js

Go-jsx is a convenient tool that converts React JSX code into Javascript without requiring Node.js.

## With Go installed

```
$ go get github.com/tiborvass/go-jsx/cmd/jsx
$ cd $GOPATH/src/github.com/tiborvass/go-jsx/examples/basic-jsx-precompile
$ $GOPATH/bin/jsx example.jsx > example.js
$ diff -u example.jsx example.js
```

# Notes

I'm not proud of the code in any way, I'm just happy it seems to work for my usecase.
Feel free to fix bugs and improve the code.

# License

The examples folder taken from [Facebook's React repository](https://github.com/facebook/react) have their own license at [examples/LICENSE-examples](./examples/LICENSE-examples).

The rest of the code is under the MIT License.
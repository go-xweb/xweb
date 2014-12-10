# xweb

Xweb is a powerful and extensiable web framework for Go. It's inspired by Struts for Java and Martini for Golang. 

[中文](https://github.com/go-xweb/xweb/blob/master/README.md)

[![Build Status](https://drone.io/github.com/go-xweb/xweb/status.png)](https://drone.io/github.com/go-xweb/xweb/latest)  [![Go Walker](http://gowalker.org/api/v1/badge)](http://gowalker.org/github.com/go-xweb/xweb)

## Changelog

* **v0.3** : All things changed. We have a new architecture inspired by struts and martini. Now you can write a interceptor yourself any time.

## Features

* Powerful routing with suburl.
* Directly integrate with existing services.
* Dynamically change template files at runtime.
* Easy to plugin/unplugin features with modular design.
* Handy dependency injection.
* simple and helpful url route mapping

## Installation

Make sure you have the a working Go environment. See the [install instructions](http://golang.org/doc/install.html). 

To install xweb, simply run:

    go get github.com/go-xweb/xweb

## Hello
```Go
package main

import "github.com/go-xweb/xweb"

type Hello struct {
}

func (Hello) Do() string {
    return "hello xweb"
}

func main() {
    xweb.AddRouter("/", new(Hello))
    xweb.Run(":8080")
}
```

## Examples

Please visit [examples](https://github.com/go-xweb/xweb/tree/master/examples) folder

## Case

* [xorm.io](http://xorm.io)
* [Godaily.org](http://godaily.org)

## Documentation

API, Please visit [GoWalker](http://gowalker.org/github.com/go-xweb/xweb)

## License
BSD License
[http://creativecommons.org/licenses/BSD/](http://creativecommons.org/licenses/BSD/)



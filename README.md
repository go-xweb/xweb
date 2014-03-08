# xweb

xweb是一个基于web.go开发的web框架，目前它和Java框架Struts有些类似。

[English](https://github.com/lunny/xweb/blob/master/README_EN.md)

[![Build Status](https://drone.io/github.com/lunny/xweb/status.png)](https://drone.io/github.com/lunny/xweb/latest)  [![Go Walker](http://gowalker.org/api/v1/badge)](http://gowalker.org/github.com/lunny/xweb) [![Bitdeli Badge](https://d2weczhvl823v0.cloudfront.net/lunny/xweb/trend.png)](https://bitdeli.com/free "Bitdeli Badge")

## 更新日志

* **v0.1.2** : 采用 [github.com/lunny/httpsession](http://github.com/lunny/httpsession) 作为session组件，API保持兼容；Action现在必须从*Action继承，这个改变与以前的版本不兼容，必须更改代码；新增两个模板函数{{session "key"}} 和 {{cookie "key"}}；Action新增函数`MapForm`
* **v0.1.1** : App新增AutoAction方法；Action新增AddTmplVar方法；Render方法的模版渲染方法中可以通过T混合传入函数和变量，更新了[快速入门](https://github.com/lunny/xweb/tree/master/docs/intro.md)。
* **v0.1.0** : 初始版本

## 特性

* 在一个可执行程序中多Server，多App的支持
* 简单好用的路由映射方式
* 静态文件及版本支持，并支持自动加载，默认开启
* 改进的模版支持，并支持自动加载，默认开启

## 安装

在安装之前确认你已经安装了Go语言. Go语言安装请访问 [install instructions](http://golang.org/doc/install.html). 

安装 xweb:

    go get github.com/lunny/xweb

## Examples

请访问 [examples](https://github.com/lunny/xweb/tree/master/examples) folder

## 案例

* [Godaily.org](http://godaily.org) - [github.com/govc/godaily](http://github.com/govc/godaily)
* [gopm.io](http://gopm.io) - [github.com/gpmgo/gopm](http://github.com/gpmgo/gopm)

## 文档

[快速入门](https://github.com/lunny/xweb/tree/master/docs/intro.md)

源码文档请访问 [GoWalker](http://gowalker.org/github.com/lunny/xweb)

## License
BSD License
[http://creativecommons.org/licenses/BSD/](http://creativecommons.org/licenses/BSD/)




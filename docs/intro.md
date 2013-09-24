#xweb介绍
xweb是一个go语言的web框架，最初的版本基于web.go，目前已经和web.go相差比较大了。

#xweb特性
1. 简单好用的路由规则
2. 静态文件支持，并支持自动加载，默认开启
3. 模版支持，并支持自动加载，默认开启

#注意事项
如果开启了静态文件自动加载和模板文件自动加载，则请确保ulimit -n调整到了一个合适的数值。

#Server
默认的Server为mainServer，这是一个全局变量。因此我们只需要调用Run即可运行服务器。

#App
每个Server都有一个默认的App，当然一个Server可以有多个App，每个App所对应到Server的路径是不同的。

#路由映射
路由映射其实有两个层次，一个层次是App层，一个App及其之下的所有Action都在某个路径之下。另一个是Action层次，每个Action都可以定义多个Mapper，一个Mapper对应一个路有规则。

Mapper的路由规则有两种形式，第一种，不使用Tag，那么这时，这个路由即为Mapper的名字，对应执行的功能为Title（mapper变量名）；第二种，使用Tag，那么这时，这个路由即为Tag语法所定义的路径。

Tag语法用空格分为如下几部分：
GET|POST		方法名，多个方法中间用|分隔
/(.*)			路径名，可以采用正则表达式
AUTO|JSON|XML   Response的方式，默认为AUTO，即自动判断，同时支持JSON和XML以及自定义方法，只要实现了xxx接口，并且进行了注册即可。

#Filter
如果要对每次请求都进行过滤，那么可以采用Filter来进行，Filter是一个接口，接口定义如下：
```Go
type Filter interface {
	Do(http.ResponseWriter, *http.Request) bool
}
```

#请求映射

通常我们通过http.Request.Form来获得从用户中请求到服务器端的数据，这些数据一般都是采用name，value的形式提交的。xweb同时提供了两种方式来获得这些数据，一种是通过Action的成员方法，GetSlice，GetString之类的；另外一种，则是通过自动映射来实现的。例如：
```Go
type HomeAction struct {
	Action
	Name string
	User User
}
```

如果在提交的表单中有一个key为name的键值对，则对应的value就会自动赋值到Name这个filed中，这种命名也可以通过.来进行传递。如：上述代码中的User结构体可以通过user.id的key来对期成员赋值。

#模板
默认采用Go语言自带的模版，同时提供了一些额外的便利方法：

#Action
Action的方法可以有不同的返回值。不同的返回值所对应的输出也不相同


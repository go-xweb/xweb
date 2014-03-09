# xweb介绍

xweb是一个基于web.go开发的web框架，目前它和Java框架Struts有些类似。


# xweb特性

* 在一个可执行程序中多Server，多App的支持
* 简单好用的路由映射方式
* 静态文件及版本支持，并支持自动加载，默认开启
* 改进的模版支持，并支持自动加载，默认开启

# 注意事项

* 如果开启了静态文件自动加载和模板文件自动加载，则请确保再启动服务之前调用过 ulimit -n 将最大文件打开数目调整到了一个合适的数值。

* 默认开启了防xsrf措施

# Server

默认的Server为mainServer，这是一个全局变量。

如果我们只需要一个server和一个app，则可以直接调用
```Go
xweb.AddAction
xweb.AutoAction
xweb.AddRouter
xweb.Run
```
即可运行服务器。

# App

每个Server都有一个默认的App，当然一个Server可以有多个App，每个App所对应到Server的路径是不同的。

# 路由映射

App路由映射其实有两个层次，一个层次是App层，一个App及其之下的所有Action都在某个路径之下。另一个是Action层次，每个Action都可以定义多个Mapper，一个Mapper对应一个路有规则。

## App路由映射

App在创建时调用`NewApp`时必须指定此app的映射规则，挂载到服务器的路径
```Go
app := xweb.NewApp("/main")
```

App在添加Action时，可以指定此App下所有Mapper的映射规则，有三个函数：

* `AddAction`
	添加一个Action的所有Mapper到App的父路径，相当于AddRouter("/", &YourAction{})

* `AutoAction`
   自动添加一个Action的所有Mapper到App的去除Action后的名称的路径，如：

```Go
type MainAction struct {
 	xweb.Action
}

app.AutoAction(&MainAction{})
/* == app.AddRouter("/main", &MainAction{})*/
```

* AddRouter
  添加一个路由到App下

## Action路由映射

Mapper的路由规则有两种形式：

* 按照Mapper的命名

这时，这个路由即为Mapper的名字，对应执行的方法名为Title（mapper变量名）；如：
```Go
type MainAction struct {
 	xweb.Action
 	
 	login xweb.Mapper
}

func (this *MainAction) Login() {

}
```
假设App和MainAction均映射为/，那么这时如果访问/login，就会执行Login方法。

* 按照Field对应的Tag语法

Tag语法用空格分为如下几部分：
GET|POST		方法名，多个方法中间用|分隔
/(.*)			路径名，可以采用正则表达式

# Filter
如果要对每次请求都进行过滤，那么可以采用Filter来进行，Filter是一个接口，接口定义如下：
```Go
type Filter interface {
	Do(http.ResponseWriter, *http.Request) bool
}
```
如果返回true，则filter不会中断会继续执行下去，如果返回false，则会中断执行，输出到浏览器。

系统已经内置了一个LoginFilter满足简单的登录需求。

# 变量映射

## 获取Form变量

通过Action的如下方法可以获取变量：

* GetSlice
* GetString
* GetInt
* GetBool
* GetFloat
* GetFile
* GetForm

## 自动映射

通常我们通过http.Request.Form来获得从用户中请求到服务器端的数据，这些数据一般都是采用name，value的形式提交的。xweb默认支持自动将这些变量映射到Action的成员中，并且这种映射支持子Struct。例如：

```Go
type HomeAction struct {
	*Action
	Name string
	User User
}
```

那么当页面为：
```Go
<form>
<input name="name"/>
<input name="user.id"/>
</form>
```
时，变量会自动映射到`HomeAction`的成员`Name`和`User`上。

如果希望关闭
那么可以在`Actions`的`Init()`方法中通过`Action.Option.AutoMapForm = false`来进行关闭。

## 手动映射

如果希望手动进行映射，那么，可以通过`Action.MapForm`方法来进行映射，例如：

```Go
type User struct {
	Id   int64
	Name string
	Age  float64
}
func (c *MainAction) Init() {
	c.Option.AutoMapForm = false
	c.Option.CheckXrsf = false
}
func (c *MainAction) Parse() error {
	if c.Method() == "GET" {
		return c.Write(page)
	} else if c.Method() == "POST" {
		var user User
		err := c.MapForm(&user, "")
		if err != nil {
			return err
		}
		return c.Write("%v", user)
	}
	return nil
}
```

如果在提交的表单中有一个key为name的键值对，则对应的value就会自动赋值到Name这个filed中，这种命名也可以通过`.`来进行传递。如：上述代码中的User结构体可以通过user.id的key来对期成员赋值。

# 模板

## 模板中的变量和函数

模板中可以使用的函数或者变量来源如下：

    1）Go模板自带的模板函数
    2）xweb内置的模板函数和变量
    3）通过Server.AddTmplVar或者AddTmplVars添加的函数或者变量
    4）通过App.AddTmplVar或者AddTmplVars添加的函数或者变量
    5）通过Action.AddTmplVar或者AddTmplVars添加的函数或者变量
    6）Action的公共变量和公共方法

## Go模板自带模板函数

这个请参见 [https://gowalker.org/text/template](https://gowalker.org/text/template)
   
## xweb内置的模板函数和变量

xweb内置的模板函数和Go模板函数一样，在模板中使用{{funcName ...}}形式调用。内置的变量使用{{.T.}}方式调用。

* `IsNil(a interface{}) bool`
判断一个指针是否为空

* `Add(left interface{}, right interface{}) interface{} `
整数或浮点数加法

* `Subtract(left interface{}, right interface{}) interface{}`
整数或者浮点数减法

* `Now()`
当前时间

* `UrlFor(args ...string) string`
返回一个route对应的url

* `FormatDate(t time.Time, format string) string`
日期格式化，格式方法和Go的格式方法相同

* `Eq(left interface{}, right interface{}) bool`
相等判断，go1.2以上请使用Go自带的eq

* `Html(raw string) template.HTML`
以html格式输出

* `StaticUrl(url string) string`
自动为静态文件添加版本标识

* `XsrfName() string`
返回xsrf的cookie名称

* `XsrfFormHtml() template.HTML`
在表单中生成防xsrf的隐藏表单

* `XsrfValue() string`
自动生成的防xsrf随机值

* `session(key string) interface{}`
获取session的值

* `cookie(key string) interface{}`
获取cookie的指


## 通过xweb.AddTmplVar或者AddTmplVars添加的函数或者变量

通过xweb.AddTmplVar或者AddTmplVars添加的函数或者变量在MainServer的RootApp范围内有效，在模板中使用{{funcName ...}}形式调用函数。变量使用{{.T.}}方式调用。

## 通过App.AddTmplVar或者AddTmplVars添加的函数或者变量

通过App.AddTmplVar或者AddTmplVars添加的函数或者变量在此App范围内有效，在模板中使用{{funcName ...}}形式调用函数。变量使用{{.T.}}方式调用。

## 通过Action.AddTmplVar或者AddTmplVars添加的函数或者变量

通过Action.AddTmplVar或者AddTmplVars添加的函数或者变量在此Action范围内有效，在模板中使用{{funcName ...}}形式调用函数。变量使用{{.T.}}方式调用。

## Action的公共变量和公共方法

Action的公共变量，通过{{.xxx}}的方式调用，公共方法通过{{.xxx}}的方式调用


## 模板包含
xweb使用include函数来进行模板包含，而不使用template函数。

* `include(tmpl string) interface{}`
包含另外一个模版

包含方式如下：

```
{{include "head.tmpl"}}
```

在包含时，只有一个模板名参数，此时head.html模板中的变量和函数也自动由当前Action传入。

## 模板事件

* BeforeRender方法
* AfterRender方法

# Action

* Init方法
* Before方法
* After方法

Action的方法可以有不同的返回值。不同的返回值所对应的输出也不相同：

1. 如果返回值为error，则检查error是否为nil，如果不为nil，则输出错误信息
2. 如果返回值为string，则将string写到body
3. 如果返回值为[]byte，则输出二进制数据。

# FAQ
1. 问：为什么post提交时会出现xsrf error的错误显示？
   答：因为默认开启了防xsrf措施，因此必须在表单中加入`{{XsrfFormHtml}}`才可以，或者将`app.AppConfig.CheckXrsf = false`


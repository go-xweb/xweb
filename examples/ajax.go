package main

import (
	"fmt"
	"reflect"

	"github.com/go-xweb/xweb"
)

var tmpl = `
<html>
<head>
<script src="https://code.jquery.com/jquery-1.11.1.min.js"></script>
<script>
function form() {
var name = $.trim($("#username").val());
    			var password = $.trim($("#password").val());
    			/*var user = {
    				username:name,
    				password:password
    			};*/

    			//var user = [name, password]
    			var user = {
    				"username":name,
    				"password":password
    			};

$.ajax({
        			url : "/login",
        			dataType : "json",
        			beforeSend: function(){
        				$(".login-btn").hide();
        				$(".login-load").show();
        			},
        			type : "post",
        			data:{"user":user},
        			success: function(data){
        				if(data.status == 0){
        					alert(data.msg);
        					$(".login-load").hide();
        					$(".login-btn").show();
        				}else{
        					$(".login-load").hide();
        					$(".login-btn").show();
        					window.location.href = "blog/list";
        				}
        			}
        		});
}

$(function(){
	$("#sub").click(form)
})
</script>
</head>
<body>
<form>
<input type="text" id="username"/>
<input type="password" id="password"/>
<input type="button" id="sub" value="登录"/>
</form>
</body>
</html>
`

type User struct {
	Username string
	Password string
}

type MainAction struct {
	*xweb.Action

	home  xweb.Mapper `xweb:"/"`
	login xweb.Mapper
	User  User
}

func (c *MainAction) Home() error {
	return c.Write(tmpl)
}

func (c *MainAction) Login() error {
	fmt.Println("user:", c.User)
	forms := c.GetForm()
	for k, v := range forms {
		fmt.Println("--", k, "-- is", reflect.TypeOf(k))
		fmt.Println("--", v, "-- is", reflect.TypeOf(v))
	}
	return nil
}

func main() {
	xweb.RootApp().AppConfig.CheckXsrf = false
	xweb.RootApp().AppConfig.SessionOn = false
	xweb.AddRouter("/", &MainAction{})
	xweb.Run("0.0.0.0:9999")
}

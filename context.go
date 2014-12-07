package xweb

type ActionContext struct {
	route     *Route
	action    interface{}
	Execute   func() interface{}
	newAction func()
}

func (ac *ActionContext) Action() interface{} {
	if ac.action == nil && ac.newAction != nil {
		ac.newAction()
	}
	return ac.action
}

func (ac *ActionContext) getRoute() *Route {
	if ac.action == nil && ac.newAction != nil {
		ac.newAction()
	}
	return ac.route
}

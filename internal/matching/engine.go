package matching

type OrderRequest struct {
	Order  *Order
	Result chan []Fill
}

type Engine struct {
	requests chan OrderRequest
	book     *Book
}

func NewEngine() *Engine {
	e := &Engine{
		requests: make(chan OrderRequest),
		book:     NewBook(),
	}
	go e.run()
	return e
}

func (e *Engine) run() {
	for req := range e.requests {
		fills := e.book.AddOrder(req.Order)
		req.Result <- fills
	}
}

func (e *Engine) Submit(o *Order) []Fill {
	result := make(chan []Fill)
	e.requests <- OrderRequest{Order: o, Result: result}
	return <-result
}
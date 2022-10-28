package main

type pact struct {
	Consumer     string
	Provider     string
	Interactions []interaction
}

type interaction struct {
	Request  request
	Response []response
}

type request struct {
	Contents contents
}
type response struct {
	Contents contents
}
type contents struct {
	Content string
}

package main

import (
	"fmt"
	"net/http"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/di"
)

type Repository struct {
}

func (r Repository) GetData() {
	fmt.Println("Calling GetData")
}

type RepositoryInterface interface {
	GetData()
}

type Service struct {
	R RepositoryInterface
}

func (s Service) Apply() {
	s.R.GetData()
	fmt.Println("Calling Apply")
}

type ServiceInterface interface {
	Apply()
}

type Controller struct {
	S ServiceInterface
}

func main() {
	// Creating an array of functions of different types
	dependencies := []any{newController, newService, newRepository}

	app := di.NewContainer()

	app.AddDependencies(dependencies)

	app.StartApp(InitializeAPP)
}

func newRepository() Repository {
	fmt.Println("Creating Repository")
	return Repository{}
}

func newService(r RepositoryInterface) Service {
	fmt.Println("Creating Service")
	return Service{
		R: r,
	}
}

func newController(s ServiceInterface) Controller {
	fmt.Println("Creating controller")
	return Controller{
		S: s,
	}
}

func (c Controller) handler(w http.ResponseWriter, r *http.Request) {
	c.S.Apply()
	fmt.Fprintf(w, "Hello, World!")
}

func InitializeAPP(c Controller) string {
	http.HandleFunc("/", c.handler)
	http.ListenAndServe(":8080", nil)
	return ""
}

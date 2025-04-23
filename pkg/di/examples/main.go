package main

import (
	"fmt"
	"net/http"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/di"
)

type Repository struct {
}

func (r Repository) GetData() {
	fmt.Println("Chamando GetData")
}

type RepositoryInterface interface {
	GetData()
}

type Service struct {
	R RepositoryInterface
}

func (s Service) Apply() {
	s.R.GetData()
	fmt.Println("Chamando Apply")
}

type ServiceInterface interface {
	Apply()
}

type Controller struct {
	S ServiceInterface
}

func main() {
	// Criação de um array de funções de diferentes tipos
	dependencies := []interface{}{newController, newService, newRepository}

	app := di.NewContainer()

	app.AddDependencies(dependencies)

	app.StartApp(InitializeAPP)
}

func newRepository() Repository {
	fmt.Println("Criando Repository")
	return Repository{}
}

func newService(r RepositoryInterface) Service {
	fmt.Println("Criando Service")
	return Service{
		R: r,
	}
}

func newController(s ServiceInterface) Controller {
	fmt.Println("Criando controller")
	return Controller{
		S: s,
	}
}

func (c Controller) handler(w http.ResponseWriter, r *http.Request) {
	c.S.Apply()
	fmt.Fprintf(w, "Olá, Mundo!")
}

func InitializeAPP(c Controller) string {
	http.HandleFunc("/", c.handler)
	http.ListenAndServe(":8080", nil)
	return ""
}

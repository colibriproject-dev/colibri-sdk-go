package di

import (
	"fmt"
	"log"
	"maps"
	"reflect"
)

type Container struct {
	dependencies map[string]DependencyBean
}

func NewContainer() Container {
	return Container{
		dependencies: make(map[string]DependencyBean),
	}
}

func (c *Container) AddDependencies(deps ...interface{}) {
	// Gera o array com as dependencias
	ReflectTypeArray := generateDependenciesArray(deps, false)
	c.checkingNameUnit(ReflectTypeArray)
	maps.Copy(c.dependencies, ReflectTypeArray)
}

func (c *Container) AddGlobalDependencies(deps ...interface{}) {
	// Gera o array com as dependencias
	ReflectTypeArray := generateDependenciesArray(deps, true)
	c.checkingNameUnit(ReflectTypeArray)
	maps.Copy(c.dependencies, ReflectTypeArray)
}

func (f *Container) StartApp(startFunc interface{}) {

	fmt.Println("Starting framework.....")
	quantDep := len(f.dependencies)
	fmt.Println(quantDep, " registered dependencies")

	dep := generateDependencyBean(startFunc, false)

	args := f.getDependencyConstructorArgs(dep)

	fmt.Println("............Starting application................")
	fmt.Println()

	// Chamando o construtor e enviando os parametros encontrados
	dep.fnValue.Call(args)

}

func (c *Container) getDependencyConstructorArgs(dependency DependencyBean) []reflect.Value {
	args := []reflect.Value{}
	fmt.Printf("constructor: %s, number of parameters: %d\n", dependency.Name, len(dependency.ParamTypes))
	for position, paramType := range dependency.ParamTypes {

		// Check if trhe variadic param
		if dependency.IsVariadic {
			if position == (len(dependency.ParamTypes) - 1) {
				// Redice slice elements to single element
				paramType = ReduceSliceToSingleElement(paramType)
			}
		}

		// Procura na lista de um contrutuores um tipo igual ao do parametro
		injectableDependencies := c.searchInjectableDependencies(paramType, dependency.constructorReturn, dependency.IsVariadic)

		for _, injectableDependency := range injectableDependencies {
			if injectableDependency.IsFunction {
				argumants := c.getDependencyConstructorArgs(injectableDependency)
				resp := injectableDependency.fnValue.Call(argumants)
				args = append(args, resp...)
				log.Println("Injecting: ", injectableDependency.Name, " in ", dependency.Name)
				if injectableDependency.IsGlobal {
					// Change function dependency to object dependency
					injectableDependency.fnValue = resp[0]
					injectableDependency.IsFunction = false
					// Update the object in the dependencies list
					c.dependencies[injectableDependency.Name] = injectableDependency
				}
			} else {
				args = append(args, injectableDependency.fnValue)
			}
		}
	}
	return args
}

func (c *Container) searchInjectableDependencies(paramType reflect.Type, returnType reflect.Type, isVariadic bool) []DependencyBean {
	var dependenciesFound []DependencyBean
	var depsFound []DependencyBean
	if isInterface(paramType) {
		dependenciesFound = c.searchImplementations(paramType)
	} else {
		dependenciesFound = c.searchTypes(paramType)
	}
	if len(dependenciesFound) > 1 {
		if isVariadic {
			depsFound = dependenciesFound
		} else {
			// O elemento 0 é o único já que os contrutores só tem um retorno
			disambiguation := searchDisambiguation(returnType, dependenciesFound)
			depsFound = append(depsFound, disambiguation)
			return depsFound
		}
	} else if len(dependenciesFound) == 0 {
		panic("nemhum construtor para o parametro foi encontrado")
	} else {
		depsFound = append(depsFound, dependenciesFound[0])
	}
	return depsFound
}

func (f *Container) searchTypes(paramType reflect.Type) []DependencyBean {
	dependenciesFound := []DependencyBean{}
	for fnName, dependency := range f.dependencies {
		for i := 0; i < dependency.constructorType.NumOut(); i++ {
			returnType := dependency.constructorType.Out(i)
			if returnType == paramType {
				fmt.Println("parameter: ", paramType, " compatible => ", fnName, " type ", returnType)
				dependenciesFound = append(dependenciesFound, dependency)
			}
		}
	}
	return dependenciesFound
}

func (f *Container) searchImplementations(paramType reflect.Type) []DependencyBean {
	dependenciesFound := []DependencyBean{}
	for fnName, dependency := range f.dependencies {
		for i := 0; i < dependency.constructorType.NumOut(); i++ {
			returnType := dependency.constructorType.Out(i)
			implements := implementsInterface(returnType, paramType)
			if implements {
				fmt.Println("parameter: ", paramType, " implementation => ", fnName, " type ", returnType)
				dependenciesFound = append(dependenciesFound, dependency)
			}
		}
	}
	return dependenciesFound
}

func (c *Container) checkingNameUnit(reflectTypeArray map[string]DependencyBean) {
	for _, v := range reflectTypeArray {
		if _, exists := c.dependencies[v.Name]; exists {
			panic("Duplicate constructor registration")
		}
	}
}

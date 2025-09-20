# Dependency Injection Container

Features of this version:

 - Automatic dependency injection	
 - Automatic identification of interface implementations	
 - **Disambiguation** via **metadata**

## About the dependency injection (DI) container	
The `di.Container` struct represents a dependency injection container and is responsible for instantiating, configuring, and assembling the components mapped in the application (beans). The container receives instructions about the components to instantiate, configure, and assemble through constructor functions and struct tags with metadata.

Basic example:

    package  main
    
    import (
    "github.com/colibriproject-dev/colibri-sdk-go/pkg/di"
    )
    
    type  Foo  struct {
    }

	func  main() {
	
	dependencies  := []any{NewFoo}
    app  :=  di.NewContainer()
    app.AddDependencies(dependencies)
    app.StartApp(InitializeAPP)
    }
    
    func InitializeAPP(f  Foo) string {
	    return  "Application started successfully!"
	}
	
	func NewFoo() Foo {
		return  Foo{}
    }

In the example above, the constructors were registered in the container using the `AddDependencies` method. After registering the dependencies, the application is started through the `StartApp` method, which receives as a parameter a function responsible for starting the entire application flow. After receiving the system initialization function, the container identifies and instantiates, via the constructor parameters, the dependencies of each object in the system.

## Fundamental concepts
### Beans

The objects that form the backbone of your application and are managed by the DI container are called beans. A bean is an object instantiated, assembled, and managed by a DI container.

All beans are built by a constructor function.

Each bean has two main properties: a name and a type.

There can be many beans of the same type, but the bean name is unique and is used to identify it.

Regarding behavior, beans can be classified into two types:

 - **Local beans** are beans that are created at injection time
 - **Global beans** are beans created once and injected into several other beans

![beans-comparation](beans-comparation.png)

The table below lists all the properties of the beans:

| Property | Description |
|--|--|
| IsFunction | Indicates whether the bean has only a constructor or an already instantiated object |
| IsGlobal | Indicates whether the bean is global or local |
| Name | The unique name of the bean |
| constructorType | Object that holds complete information about the constructor |
| fnValue | Object that holds the constructor to be invoked for building the object |
| constructorReturn | Object that holds the exact type of the constructor, used to obtain metadata |
| ParamTypes | The parameters for building the bean |


### Bean constructors

Constructors are functions responsible for creating beans.

Bean constructors can only have 1 return value, which is the bean itself.

Bean constructors must either receive other beans as parameters or receive no parameters (root constructors)

### Disambiguation

During the mapping and injection process, if more than one constructor is found for a bean, tag metadata is used to determine which one should be injected.

    type  BeanWithMetadata  struct {
    	f  BeanDependency  `di:"NewBeanDependency2"`
    }
    
    func  NewBeanDependency1() BeanDependency {
    	return  BeanDependency{}
    }
    
    func  NewBeanDependency2() BeanDependency {
    	return  BeanDependency{}
    }  

> **Note:** disambiguation does not work on variadic parameters
  
## Container operation flow

The dependency injection container works through a process of stacking and unstacking (injection). In the stacking phase, the dependencies of an object and the dependencies of those dependencies are identified, in a recursive cycle that ends when objects that do not require dependency injection are found. In the injection phase, the objects mapped in the stack are created so that the objects at the top of the stack are used as parameters to create the objects in the lower layers.

![flow](flow-1.png)

1. Register the constructors responsible for creating all the application dependencies. The dependencies created by the constructors and injected into other constructors' parameters are called beans.

2. Register the function responsible for starting the entire application flow.

3. Identify the beans that this function receives as parameters.

4. Look up in the constructor registry the constructors for those beans.

	1. If these constructors also receive other beans as parameters, a recursive cycle of bean lookup and constructor identification will start.

	2. This cycle ends when constructors that do not receive parameters (root constructors) or a global bean are found.

	3. If more than one constructor for a bean is found, tag metadata is used to determine which should be injected.

5. When the root beans are found (those that have no parameters), the recursion ends and the process of object construction begins.
package di

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

func isInterface(r reflect.Type) bool {
	return r.Kind() == reflect.Interface
}

func searchDisambiguation(returnType reflect.Type, dependenciesFound []DependencyBean) DependencyBean {
	// Iterate over struct fields and read metadata
	tags := getTagsInType(returnType, "di")
	for fieldName, tagValue := range tags {
		for _, dependency := range dependenciesFound {
			nameParts := strings.Split(dependency.Name, ".")
			if nameParts[len(nameParts)-1] == tagValue {
				fmt.Printf("Disambiguation: METADATA in %v on %s = %s", returnType, fieldName, tagValue)
				return dependency
			}
		}
	}
	panic("More than one constructor found for the same type, no METADATA found to resolve the ambiguity")
	return DependencyBean{}
}

// Checks whether a struct implements an interface
func implementsInterface(structType reflect.Type, interfaceType reflect.Type) bool {
	return structType.Implements(interfaceType)
}

func generateDependenciesArray(funcs []any, isGlobal bool) map[string]DependencyBean {
	ReflectTypeArray := make(map[string]DependencyBean)
	for _, fn := range funcs {
		dep := generateDependencyBean(fn, isGlobal)
		ReflectTypeArray[dep.Name] = dep
	}
	return ReflectTypeArray
}

func getFunctionName(i reflect.Value) string {
	return runtime.FuncForPC(i.Pointer()).Name()
}

func getParamTypes(fnType reflect.Type) []reflect.Type {
	var paramTypes []reflect.Type
	for i := 0; i < fnType.NumIn(); i++ {
		paramTypes = append(paramTypes, fnType.In(i))
	}
	return paramTypes
}

func getReturnType(fnType reflect.Type) reflect.Type {
	if fnType.NumOut() == 1 {
		return fnType.Out(0)
	} else {
		message := fmt.Sprintf("Error, the function %s must have a single return type\n", fnType.Name())
		panic(message)
	}
}

func generateDependencyBean(fn any, isGlobal bool) DependencyBean {
	fnType := reflect.TypeOf(fn)
	fnValue := reflect.ValueOf(fn)
	nameFunction := getFunctionName(fnValue)
	paramTypes := getParamTypes(fnType)
	returnType := getReturnType(fnType)
	isVariadic := fnType.IsVariadic()
	return DependencyBean{
		constructorType:   fnType,
		fnValue:           fnValue,
		Name:              nameFunction,
		IsGlobal:          isGlobal,
		IsFunction:        true,
		IsVariadic:        isVariadic,
		constructorReturn: returnType,
		ParamTypes:        paramTypes}
}

func getTagsInType(objectType reflect.Type, tagName string) map[string]string {
	tags := make(map[string]string)
	numField := objectType.NumField()
	if numField == 0 {
		message := fmt.Sprintf("struct %v with more than one constructor and no values to disqualify", objectType)
		panic(message)
	}
	for i := 0; i < numField; i++ {
		field := objectType.Field(i)
		// get tag metadata
		tagValue := field.Tag.Get(tagName)
		fmt.Println("tagValue: ", tagValue)
		tags[field.Name] = tagValue
	}
	return tags
}

func ReduceSliceToSingleElement(sliceElem reflect.Type) reflect.Type {
	if sliceElem.Kind() == reflect.Slice {
		elementType := sliceElem.Elem()
		return elementType
	}
	return sliceElem
}

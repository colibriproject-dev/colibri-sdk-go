# Changelog

## Version 2024-06-28

In this version:
- The return of the `searchInjectableDependencies` function was changed to a []DependencyBean.
- An `isVariadic` parameter was added to the `searchInjectableDependencies` function; this parameter reduces or not the number of results found.
- Logic was added to obtain the unit value of slices in variadic parameters.
- The call of constructors with variadic parameters was added.
- A note was added about disambiguation not working on variadic parameters.
- The `isVariadic` field was added to the `DependencyBean` struct.
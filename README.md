# TypeScript Compiler with Golang and Goja Experiment

**This was hacked together in about 3-4 hours**

An experiment to see if the TypeScript compiler can be embedded into a Golang application using Goja as the JavaScript interpreter/VM.

This [main.ts](testdata/.fel/main.ts) file is attempted to be compiled, ie.
```ts
function test(a: number) {
	return "hey"
}

test("test");
```

The current output message is currently:
```
testdata/.fel/main.ts:Argument of type '"test"' is not assignable to parameter of type 'number'.

Total time taken: 16.5468106s
panic: end

goroutine 1 [running]:
main.main()
        D:/GoProjectsModules/compile-typescript/main.go:210 +0xd3b
```

As you can see, the current compile-time is huge for a single file. A suspect because the TypeScript compiler is currently spending the bulk of it's time parsing the type definition files found in the [typescriptServices](typescriptServices/v3.4.5) folder.
